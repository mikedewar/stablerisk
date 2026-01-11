package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// OutlierType represents the type of anomaly detected
type OutlierType string

const (
	OutlierTypeZScore              OutlierType = "zscore"
	OutlierTypeIQR                 OutlierType = "iqr"
	OutlierTypePatternCirculation  OutlierType = "pattern_circulation"
	OutlierTypePatternFanOut       OutlierType = "pattern_fanout"
	OutlierTypePatternFanIn        OutlierType = "pattern_fanin"
	OutlierTypePatternDormant      OutlierType = "pattern_dormant"
	OutlierTypePatternVelocity     OutlierType = "pattern_velocity"
)

// Severity represents the severity level of an outlier
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// Outlier represents a detected anomaly
type Outlier struct {
	ID              string          `json:"id"`
	DetectedAt      time.Time       `json:"detected_at"`
	Type            OutlierType     `json:"type"`
	Severity        Severity        `json:"severity"`
	Address         string          `json:"address"`
	TransactionHash string          `json:"transaction_hash,omitempty"`
	Amount          decimal.Decimal `json:"amount,omitempty"`
	ZScore          float64         `json:"z_score,omitempty"`
	Details         map[string]interface{} `json:"details"`
	Acknowledged    bool            `json:"acknowledged"`
	AcknowledgedBy  string          `json:"acknowledged_by,omitempty"`
	AcknowledgedAt  time.Time       `json:"acknowledged_at,omitempty"`
	Notes           string          `json:"notes,omitempty"`
}

// StatisticalData holds statistical information for anomaly detection
type StatisticalData struct {
	Values []float64
	Mean   float64
	StdDev float64
	Q1     float64
	Q2     float64 // Median
	Q3     float64
	IQR    float64
	Min    float64
	Max    float64
	Count  int
}

// AddressActivity represents transaction activity for an address
type AddressActivity struct {
	Address         string
	TransactionCount int
	SentCount       int
	ReceivedCount   int
	TotalSent       decimal.Decimal
	TotalReceived   decimal.Decimal
	FirstSeen       time.Time
	LastSeen        time.Time
	Neighbors       []string
}

// PatternMatch represents a detected pattern
type PatternMatch struct {
	PatternType string
	Addresses   []string
	Transactions []string
	Confidence  float64
	Description string
}
