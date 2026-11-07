package config

import (
	"fmt"
	"os"
	"strings"

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

	// HTTP server settings
	Server ServerConfig `mapstructure:"server"`

	// Security settings (shared by CORS and the WebSocket Origin check)
	Security SecurityConfig `mapstructure:"security"`

	// Adapter settings
	Adapter AdapterConfig `mapstructure:"adapter"`
}

// PlaceholderJWTSecret is the value shipped in config/config.yaml and used as
// the viper default. A deployment that never changed it signs tokens with a
// public key, so startup warns when it is still in place.
const PlaceholderJWTSecret = "change-this-secret-in-production"

// DefaultAllowedOrigins is the development Origin allowlist. It covers the Vite
// dev server and the backend's own default port; a deployment adds its real
// front-end Origin via config.yaml or the ESP_SECURITY_ALLOWED_ORIGINS env var.
func DefaultAllowedOrigins() []string {
	return []string{
		"http://localhost:3000",
		"http://127.0.0.1:3000",
		"http://localhost:8080",
		"http://127.0.0.1:8080",
	}
}

// ServerConfig holds HTTP server timeouts and request-size limits.
type ServerConfig struct {
	// ReadHeaderTimeout bounds how long a client may take to send headers.
	ReadHeaderTimeout int `mapstructure:"read_header_timeout"` // seconds
	// ReadTimeout bounds the whole request read.
	ReadTimeout int `mapstructure:"read_timeout"` // seconds
	// WriteTimeout bounds the response write.
	WriteTimeout int `mapstructure:"write_timeout"` // seconds
	// IdleTimeout bounds an idle keep-alive connection.
	IdleTimeout int `mapstructure:"idle_timeout"` // seconds
	// MaxBodyBytes caps the request body size.
	MaxBodyBytes int64 `mapstructure:"max_body_bytes"` // bytes
}

// SecurityConfig holds cross-cutting browser-security settings.
type SecurityConfig struct {
	// AllowedOrigins is the Origin allowlist used both by the CORS middleware
	// and by the WebSocket CheckOrigin hook. One list, so the two can never
	// drift apart.
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

// IsAllowedOrigin reports whether origin is in the CORS/WebSocket allowlist.
// The empty Origin (non-browser clients such as the Python bridges) is allowed,
// as is an empty allowlist meaning "development default", i.e. allow all.
func (c *Config) IsAllowedOrigin(origin string) bool {
	if origin == "" {
		return true
	}
	allowed := c.Security.AllowedOrigins
	if len(allowed) == 0 {
		return true
	}
	for _, a := range allowed {
		if a == origin {
			return true
		}
	}
	return false
}

// UsingPlaceholderJWTSecret reports whether the JWT secret is still the
// shipped placeholder.
func (c *Config) UsingPlaceholderJWTSecret() bool {
	return c.JWT.Secret == "" || c.JWT.Secret == PlaceholderJWTSecret
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

	// Python adapter process management
	PythonBin   string `mapstructure:"python_bin"`   // python executable path (default: python3)
	AdaptersDir string `mapstructure:"adapters_dir"` // root dir containing adapters/<platform>/main.py (default: adapters)
	AutoRestart bool   `mapstructure:"auto_restart"` // auto-restart crashed python processes (default: true)
}

// FormOptionsPath returns the path to the form options YAML registry file.
// The file is resolved relative to the current working directory and the
// ./config directory, mirroring how the main config file is discovered.
func (c *Config) FormOptionsPath() string {
	candidates := []string{
		"config/form-options.yaml",
		"./config/form-options.yaml",
		"/etc/e-sp-line2/form-options.yaml",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	// Fallback: default to the config directory location
	return "config/form-options.yaml"
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
	v.SetDefault("jwt.secret", PlaceholderJWTSecret)
	v.SetDefault("jwt.expiration", 24)

	// HTTP server defaults. Without these a single slow client can hold a
	// connection forever (Slowloris) and a large body is read entirely into
	// memory.
	v.SetDefault("server.read_header_timeout", 10)
	v.SetDefault("server.read_timeout", 60)
	v.SetDefault("server.write_timeout", 120)
	v.SetDefault("server.idle_timeout", 120)
	v.SetDefault("server.max_body_bytes", 4<<20) // 4 MiB

	// Origin allowlist shared by CORS and the WebSocket Origin check.
	v.SetDefault("security.allowed_origins", DefaultAllowedOrigins())

	// Env is read with ESP_* names; AutomaticEnv alone cannot see a slice, so
	// the comma-separated override is bound explicitly. This lets a deployment
	// set its real front-end Origin without editing config.yaml.
	v.SetEnvPrefix("esp")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	_ = v.BindEnv("security.allowed_origins")

	// Adapter defaults
	v.SetDefault("adapter.heartbeat_interval", 30)
	v.SetDefault("adapter.reconnect_delay", 5)
	v.SetDefault("adapter.max_retries", 3)
	v.SetDefault("adapter.message_queue_size", 1000)

	// Python adapter process defaults
	v.SetDefault("adapter.python_bin", "python3")
	v.SetDefault("adapter.adapters_dir", "adapters")
	v.SetDefault("adapter.auto_restart", true)
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
