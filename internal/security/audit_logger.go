package security

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        string
	Timestamp time.Time
	UserID    string
	Action    string
	Resource  string
	Status    string
	IPAddress string
	Details   map[string]interface{}
	Signature string
}

// AuditLogger handles tamper-proof audit logging
type AuditLogger struct {
	db         *sql.DB
	secretKey  []byte
	logger     *zap.Logger
	logChan    chan *AuditLog
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	batchSize  int
	flushInterval time.Duration
}

// AuditLoggerConfig holds configuration for audit logger
type AuditLoggerConfig struct {
	SecretKey     string
	BatchSize     int           // Number of logs to batch before writing
	FlushInterval time.Duration // Maximum time to wait before flushing
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(db *sql.DB, config AuditLoggerConfig, logger *zap.Logger) *AuditLogger {
	if logger == nil {
		logger = zap.NewNop()
	}

	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}

	if config.FlushInterval <= 0 {
		config.FlushInterval = 5 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	al := &AuditLogger{
		db:            db,
		secretKey:     []byte(config.SecretKey),
		logger:        logger,
		logChan:       make(chan *AuditLog, 1000),
		ctx:           ctx,
		cancel:        cancel,
		batchSize:     config.BatchSize,
		flushInterval: config.FlushInterval,
	}

	// Start worker to process audit logs
	al.wg.Add(1)
	go al.worker()

	return al
}

// Log creates an audit log entry
func (al *AuditLogger) Log(userID, action, resource, status, ipAddress string, details map[string]interface{}) {
	log := &AuditLog{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		UserID:    userID,
		Action:    action,
		Resource:  resource,
		Status:    status,
		IPAddress: ipAddress,
		Details:   details,
	}

	// Generate HMAC signature for tamper-proofing
	log.Signature = al.generateSignature(log)

	// Send to channel (non-blocking)
	select {
	case al.logChan <- log:
	default:
		al.logger.Error("Audit log channel full, dropping log entry",
			zap.String("action", action),
			zap.String("user_id", userID))
	}
}

// worker processes audit logs from the channel
func (al *AuditLogger) worker() {
	defer al.wg.Done()

	batch := make([]*AuditLog, 0, al.batchSize)
	ticker := time.NewTicker(al.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case log := <-al.logChan:
			batch = append(batch, log)
			if len(batch) >= al.batchSize {
				al.flushBatch(batch)
				batch = make([]*AuditLog, 0, al.batchSize)
			}

		case <-ticker.C:
			if len(batch) > 0 {
				al.flushBatch(batch)
				batch = make([]*AuditLog, 0, al.batchSize)
			}

		case <-al.ctx.Done():
			// Flush remaining logs before shutdown
			if len(batch) > 0 {
				al.flushBatch(batch)
			}
			// Drain channel
			for {
				select {
				case log := <-al.logChan:
					al.flushBatch([]*AuditLog{log})
				default:
					return
				}
			}
		}
	}
}

// flushBatch writes a batch of audit logs to the database
func (al *AuditLogger) flushBatch(logs []*AuditLog) {
	if len(logs) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := al.db.BeginTx(ctx, nil)
	if err != nil {
		al.logger.Error("Failed to begin transaction for audit logs",
			zap.Error(err))
		return
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO audit_logs (id, timestamp, user_id, action, resource, status, ip_address, details, signature)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`)
	if err != nil {
		tx.Rollback()
		al.logger.Error("Failed to prepare audit log statement",
			zap.Error(err))
		return
	}
	defer stmt.Close()

	for _, log := range logs {
		detailsJSON, err := json.Marshal(log.Details)
		if err != nil {
			al.logger.Error("Failed to marshal audit log details",
				zap.Error(err),
				zap.String("log_id", log.ID))
			continue
		}

		_, err = stmt.ExecContext(ctx,
			log.ID,
			log.Timestamp,
			nullString(log.UserID),
			log.Action,
			nullString(log.Resource),
			log.Status,
			nullString(log.IPAddress),
			detailsJSON,
			log.Signature,
		)
		if err != nil {
			al.logger.Error("Failed to insert audit log",
				zap.Error(err),
				zap.String("log_id", log.ID))
		}
	}

	if err := tx.Commit(); err != nil {
		al.logger.Error("Failed to commit audit logs",
			zap.Error(err))
	} else {
		al.logger.Debug("Flushed audit logs",
			zap.Int("count", len(logs)))
	}
}

// generateSignature generates an HMAC-SHA256 signature for the audit log
func (al *AuditLogger) generateSignature(log *AuditLog) string {
	// Create a canonical representation of the log
	detailsJSON, _ := json.Marshal(log.Details)
	data := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s",
		log.ID,
		log.Timestamp.Format(time.RFC3339Nano),
		log.UserID,
		log.Action,
		log.Resource,
		log.Status,
		log.IPAddress,
		string(detailsJSON),
	)

	// Generate HMAC
	h := hmac.New(sha256.New, al.secretKey)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifySignature verifies the HMAC signature of an audit log
func (al *AuditLogger) VerifySignature(log *AuditLog) bool {
	expectedSignature := al.generateSignature(log)
	return hmac.Equal([]byte(expectedSignature), []byte(log.Signature))
}

// Close gracefully shuts down the audit logger
func (al *AuditLogger) Close() error {
	al.logger.Info("Shutting down audit logger")
	al.cancel()
	al.wg.Wait()
	al.logger.Info("Audit logger shutdown complete")
	return nil
}

// nullString returns sql.NullString for empty strings
func nullString(s string) sql.NullString {
	return sql.NullString{
		String: s,
		Valid:  s != "",
	}
}
