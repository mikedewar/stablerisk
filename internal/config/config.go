package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	TronGrid   TronGridConfig   `mapstructure:"trongrid"`
	Raphtory   RaphtoryConfig   `mapstructure:"raphtory"`
	Security   SecurityConfig   `mapstructure:"security"`
	Detection  DetectionConfig  `mapstructure:"detection"`
	Logging    LoggingConfig    `mapstructure:"logging"`
	Monitoring MonitoringConfig `mapstructure:"monitoring"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	APIPort        int           `mapstructure:"api_port"`
	ReadTimeout    time.Duration `mapstructure:"read_timeout"`
	WriteTimeout   time.Duration `mapstructure:"write_timeout"`
	MaxHeaderBytes int           `mapstructure:"max_header_bytes"`
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// TronGridConfig holds TronGrid API configuration
type TronGridConfig struct {
	APIKey          string        `mapstructure:"api_key"`
	WebSocketURL    string        `mapstructure:"websocket_url"` // Actually REST API URL (https://), kept for backwards compat
	USDTContract    string        `mapstructure:"usdt_contract"`
	ReconnectDelay  time.Duration `mapstructure:"reconnect_delay"`
	MaxReconnects   int           `mapstructure:"max_reconnects"`
	PingInterval    time.Duration `mapstructure:"ping_interval"` // Used as polling interval for REST API
}

// RaphtoryConfig holds Raphtory service configuration
type RaphtoryConfig struct {
	BaseURL        string        `mapstructure:"base_url"`
	Timeout        time.Duration `mapstructure:"timeout"`
	MaxRetries     int           `mapstructure:"max_retries"`
	RetryDelay     time.Duration `mapstructure:"retry_delay"`
}

// SecurityConfig holds security and compliance configuration
type SecurityConfig struct {
	JWTSecret           string        `mapstructure:"jwt_secret"`
	JWTExpiry           time.Duration `mapstructure:"jwt_expiry"`
	RefreshTokenExpiry  time.Duration `mapstructure:"refresh_token_expiry"`
	EncryptionKey       string        `mapstructure:"encryption_key"`
	HMACKey             string        `mapstructure:"hmac_key"`
	TLSEnabled          bool          `mapstructure:"tls_enabled"`
	TLSCertFile         string        `mapstructure:"tls_cert_file"`
	TLSKeyFile          string        `mapstructure:"tls_key_file"`
	PasswordMinLength   int           `mapstructure:"password_min_length"`
	PasswordHashCost    int           `mapstructure:"password_hash_cost"`
}

// DetectionConfig holds anomaly detection configuration
type DetectionConfig struct {
	Interval             time.Duration `mapstructure:"interval"`
	ZScoreThreshold      float64       `mapstructure:"zscore_threshold"`
	IQRMultiplier        float64       `mapstructure:"iqr_multiplier"`
	WindowDuration       time.Duration `mapstructure:"window_duration"`
	MinDataPoints        int           `mapstructure:"min_data_points"`
	PatternDetectionEnabled bool       `mapstructure:"pattern_detection_enabled"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	OutputPath string `mapstructure:"output_path"`
	ErrorPath  string `mapstructure:"error_path"`
}

// MonitoringConfig holds monitoring and observability configuration
type MonitoringConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	MetricsPort    int    `mapstructure:"metrics_port"`
	HealthCheckURL string `mapstructure:"health_check_url"`
}

// Load reads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set default values
	setDefaults(v)

	// Configure environment variable handling BEFORE reading config file
	v.AutomaticEnv()
	v.SetEnvPrefix("STABLERISK")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set config file path
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./internal/config")
		v.AddConfigPath(".")
	}

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found; use defaults and env vars
	}

	// Unmarshal config
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.api_port", 8080)
	v.SetDefault("server.read_timeout", 10*time.Second)
	v.SetDefault("server.write_timeout", 10*time.Second)
	v.SetDefault("server.max_header_bytes", 1<<20) // 1 MB

	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "stablerisk")
	v.SetDefault("database.database", "stablerisk")
	v.SetDefault("database.ssl_mode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", 5*time.Minute)

	// TronGrid defaults
	// Note: websocket_url is now used for REST API (https://), not WebSocket (wss://)
	v.SetDefault("trongrid.websocket_url", "https://api.trongrid.io")
	v.SetDefault("trongrid.usdt_contract", "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	v.SetDefault("trongrid.reconnect_delay", 1*time.Second)
	v.SetDefault("trongrid.max_reconnects", 10)
	v.SetDefault("trongrid.ping_interval", 10*time.Second) // Used as polling interval

	// Raphtory defaults
	v.SetDefault("raphtory.base_url", "http://localhost:8000")
	v.SetDefault("raphtory.timeout", 30*time.Second)
	v.SetDefault("raphtory.max_retries", 3)
	v.SetDefault("raphtory.retry_delay", 1*time.Second)

	// Security defaults
	v.SetDefault("security.jwt_expiry", 1*time.Hour)
	v.SetDefault("security.refresh_token_expiry", 7*24*time.Hour)
	v.SetDefault("security.tls_enabled", false)
	v.SetDefault("security.password_min_length", 12)
	v.SetDefault("security.password_hash_cost", 12)

	// Detection defaults
	v.SetDefault("detection.interval", 60*time.Second)
	v.SetDefault("detection.zscore_threshold", 3.0)
	v.SetDefault("detection.iqr_multiplier", 1.5)
	v.SetDefault("detection.window_duration", 24*time.Hour)
	v.SetDefault("detection.min_data_points", 30)
	v.SetDefault("detection.pattern_detection_enabled", true)

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.output_path", "stdout")
	v.SetDefault("logging.error_path", "stderr")

	// Monitoring defaults
	v.SetDefault("monitoring.enabled", true)
	v.SetDefault("monitoring.metrics_port", 9090)
	v.SetDefault("monitoring.health_check_url", "/health")
}

// validate checks if the configuration is valid
func validate(cfg *Config) error {
	// Validate TronGrid API key
	if cfg.TronGrid.APIKey == "" {
		return fmt.Errorf("trongrid.api_key is required")
	}

	// Validate USDT contract address
	if cfg.TronGrid.USDTContract == "" {
		return fmt.Errorf("trongrid.usdt_contract is required")
	}

	// Validate security keys
	if cfg.Security.JWTSecret == "" {
		return fmt.Errorf("security.jwt_secret is required")
	}
	if cfg.Security.EncryptionKey == "" {
		return fmt.Errorf("security.encryption_key is required")
	}
	if cfg.Security.HMACKey == "" {
		return fmt.Errorf("security.hmac_key is required")
	}

	// Validate database password
	if cfg.Database.Password == "" {
		return fmt.Errorf("database.password is required")
	}

	// Validate detection thresholds
	if cfg.Detection.ZScoreThreshold <= 0 {
		return fmt.Errorf("detection.zscore_threshold must be positive")
	}
	if cfg.Detection.IQRMultiplier <= 0 {
		return fmt.Errorf("detection.iqr_multiplier must be positive")
	}

	return nil
}
