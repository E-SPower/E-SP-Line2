package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	// Server settings
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
	Version string `mapstructure:"version"`

	// Database settings
	Database DatabaseConfig `mapstructure:"database"`

	// Redis settings
	Redis RedisConfig `mapstructure:"redis"`

	// Log settings
	LogLevel string `mapstructure:"log_level"`

	// JWT settings
	JWT JWTConfig `mapstructure:"jwt"`

	// Adapter settings
	Adapter AdapterConfig `mapstructure:"adapter"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Driver   string `mapstructure:"driver"`
	DSN      string `mapstructure:"dsn"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// JWTConfig represents JWT configuration
type JWTConfig struct {
	Secret     string `mapstructure:"secret"`
	Expiration int    `mapstructure:"expiration"` // in hours
}

// AdapterConfig represents adapter configuration
type AdapterConfig struct {
	HeartbeatInterval int `mapstructure:"heartbeat_interval"` // in seconds
	ReconnectDelay    int `mapstructure:"reconnect_delay"`    // in seconds
	MaxRetries        int `mapstructure:"max_retries"`
	MessageQueueSize  int `mapstructure:"message_queue_size"`
}

// Load loads the configuration from file and environment
func Load() (*Config, error) {
	v := viper.New()

	// Set config file name and path
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/e-sp-line2")

	// Set default values
	setDefaults(v)

	// Bind environment variables
	v.AutomaticEnv()

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found, use defaults and env vars
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("host", "0.0.0.0")
	v.SetDefault("port", 8080)
	v.SetDefault("version", "1.0.0")

	// Database defaults
	v.SetDefault("database.driver", "sqlite")
	v.SetDefault("database.dsn", "data/e-sp-line2.db")
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "esp")
	v.SetDefault("database.password", "")
	v.SetDefault("database.dbname", "esp_line2")
	v.SetDefault("database.sslmode", "disable")

	// Redis defaults
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	// Log defaults
	v.SetDefault("log_level", "info")

	// JWT defaults
	v.SetDefault("jwt.secret", "change-this-secret-in-production")
	v.SetDefault("jwt.expiration", 24)

	// Adapter defaults
	v.SetDefault("adapter.heartbeat_interval", 30)
	v.SetDefault("adapter.reconnect_delay", 5)
	v.SetDefault("adapter.max_retries", 3)
	v.SetDefault("adapter.message_queue_size", 1000)
}

// GetDSN returns the database connection string
func (c *DatabaseConfig) GetDSN() string {
	if c.DSN != "" {
		return c.DSN
	}

	switch c.Driver {
	case "postgres":
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
	case "sqlite":
		return c.DSN
	default:
		return c.DSN
	}
}

// GetRedisAddr returns the Redis address
func (c *RedisConfig) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
