package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// PlaceholderJWTSecret is the value shipped in config/config.yaml and used as
// the viper default. A deployment that still runs with it signs tokens with a
// secret anybody can read in the repository, so startup warns about it.
const PlaceholderJWTSecret = "change-this-secret-in-production"

// Config represents the application configuration
type Config struct {
	// Server settings
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
	Version string `mapstructure:"version"`

	// HTTP hardening, plus the browser origins allowed to call the API and the
	// WebSocket handshake. See ServerConfig and CORSConfig.
	Server ServerConfig `mapstructure:"server"`

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

// ServerConfig holds the HTTP server hardening knobs and the browser origin
// allow-list.
//
// The timeouts are seconds, matching the rest of the config file, and are
// converted to time.Duration in server.New. A zero value disables the
// corresponding net/http timeout, which is what the standard library does too.
type ServerConfig struct {
	// AllowedOrigins is the allow-list shared by CORS and the WebSocket
	// handshake, so the two never drift apart. An entry of "*" allows any
	// origin; otherwise the Origin header is echoed only when it matches an
	// entry exactly (scheme + host + port, no trailing slash).
	AllowedOrigins []string `mapstructure:"allowed_origins"`

	ReadHeaderTimeout int `mapstructure:"read_header_timeout"` // seconds
	ReadTimeout       int `mapstructure:"read_timeout"`        // seconds
	WriteTimeout      int `mapstructure:"write_timeout"`       // seconds
	IdleTimeout       int `mapstructure:"idle_timeout"`        // seconds

	// MaxBodyBytes caps an HTTP request body. Uploads are JSON blobs (adapter
	// configs, cookies), so this stays far below a file-transfer size.
	MaxBodyBytes int64 `mapstructure:"max_body_bytes"`
}

// OriginAllowed reports whether origin is in the allow-list. A "*" entry
// allows everything.
func (c *ServerConfig) OriginAllowed(origin string) bool {
	for _, allowed := range c.AllowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
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
	NodeBin     string `mapstructure:"node_bin"`     // node executable path for standalone adapters (default: node)
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

// JWTSecretWarning returns the warning a caller should log when secret is
// still the shipped placeholder (or empty), and "" when it was changed.
//
// The check is a pure function, not a log call inside Load, because Load runs
// before logger.Init and a nil zap logger would silently drop it.
func JWTSecretWarning(secret string) string {
	switch secret {
	case PlaceholderJWTSecret:
		return "jwt.secret is still the shipped placeholder " + PlaceholderJWTSecret +
			"; anyone with the repository can forge a token for this deployment. " +
			"Set jwt.secret in config/config.yaml, or export JWT.SECRET to override it."
	case "":
		// An empty key still signs, with nothing. Reached by exporting
		// JWT.SECRET= or blanking the key, not by the shipped config.
		return "jwt.secret is empty; tokens are signed with an empty HS256 key. " +
			"Set jwt.secret in config/config.yaml."
	default:
		return ""
	}
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("host", "0.0.0.0")
	v.SetDefault("port", 8080)
	v.SetDefault("version", "1.0.0")

	// HTTP hardening. 5s is enough for every header block a browser sends and
	// still ends a Slowloris connection quickly; the read/write budgets cover a
	// large JSON body; idle covers keep-alive reuse.
	v.SetDefault("server.read_header_timeout", 5)
	v.SetDefault("server.read_timeout", 60)
	v.SetDefault("server.write_timeout", 60)
	v.SetDefault("server.idle_timeout", 120)
	v.SetDefault("server.max_body_bytes", 8<<20) // 8 MiB

	// Origins allowed to call the API and open a WebSocket. The Vite dev server
	// is the only cross-origin caller out of the box; a deployment behind a
	// reverse proxy is same-origin and needs no entry. Replace/add entries
	// rather than widening to "*", see docs/developer-guide/api-reference.md.
	v.SetDefault("server.allowed_origins", []string{
		"http://localhost:3000",
		"http://127.0.0.1:3000",
	})

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

	// Adapter defaults
	v.SetDefault("adapter.heartbeat_interval", 30)
	v.SetDefault("adapter.reconnect_delay", 5)
	v.SetDefault("adapter.max_retries", 3)
	v.SetDefault("adapter.message_queue_size", 1000)

	// Python adapter process defaults
	v.SetDefault("adapter.python_bin", "python3")
	v.SetDefault("adapter.node_bin", "node")
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
