package config

import (
	"time"
)

type Config struct {
	Auth           SharedAuthConfig     `yaml:"auth"`
	Database       DatabaseConfig       `yaml:"database"`
	DLQ            DLQConfig            `yaml:"dlq"`
	QueryInterface QueryInterfaceConfig `yaml:"query-interface"`
}

// SharedAuthConfig holds the jwt secret both services validate tokens against.
// only query-interface issues tokens (see /auth/login); log-ingestor only validates them.
type SharedAuthConfig struct {
	JWTSecret string `yaml:"jwtSecret"`
}

type DatabaseConfig struct {
	Postgres PostgresConfig `yaml:"postgres"`
}

type PostgresConfig struct {
	Host           string               `yaml:"host"`
	Port           int                  `yaml:"port"`
	User           string               `yaml:"user"`
	Password       string               `yaml:"password"`
	DBName         string               `yaml:"dbname"`
	ConnectionPool ConnectionPoolConfig `yaml:"connectionPool"`
}

type ConnectionPoolConfig struct {
	MaxOpenConns    int           `yaml:"maxOpenConns"`
	MaxIdleConns    int           `yaml:"maxIdleConns"`
	ConnMaxLifetime time.Duration `yaml:"connMaxLifetime"`
	ConnMaxIdleTime time.Duration `yaml:"connMaxIdleTime"`
}

type DLQConfig struct {
	Enabled      bool                `yaml:"enabled"`
	Type         string              `yaml:"type"`
	Connection   DLQConnectionConfig `yaml:"connection"`
	Concurrency  int                 `yaml:"concurrency"`
	FailureTypes FailureTypesConfig  `yaml:"failureTypes"`
}

type DLQConnectionConfig struct {
	URI        string `yaml:"uri"`
	Database   string `yaml:"database"`
	Collection string `yaml:"collection"`
}

type FailureTypesConfig struct {
	ParseError         bool `yaml:"parseError"`
	DBFailure          bool `yaml:"dbFailure"`
	ValidationError    bool `yaml:"validationError"`
	ElasticsearchError bool `yaml:"elasticsearchError"`
	TimeoutError       bool `yaml:"timeoutError"`
}

type QueryInterfaceConfig struct {
	Server        ServerConfig        `yaml:"server"`
	Elasticsearch ElasticsearchConfig `yaml:"elasticsearch"`
	Search        SearchConfig        `yaml:"search"`
	Auth          AuthConfig          `yaml:"auth"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type ElasticsearchConfig struct {
	Connection ElasticsearchConnectionConfig `yaml:"connection"`
	Indexer    ElasticsearchIndexerConfig    `yaml:"indexer"`
}

type ElasticsearchConnectionConfig struct {
	Host           string                    `yaml:"host"`
	Index          string                    `yaml:"index"`
	Timeout        time.Duration             `yaml:"timeout"`
	ConnectionPool ElasticsearchPoolConfig   `yaml:"connectionPool"`
	HealthCheck    ElasticsearchHealthConfig `yaml:"healthCheck"`
}

type ElasticsearchPoolConfig struct {
	MaxConnsPerHost int           `yaml:"maxConnsPerHost"`
	MaxIdleConns    int           `yaml:"maxIdleConns"`
	IdleConnTimeout time.Duration `yaml:"idleConnTimeout"`
}

type ElasticsearchHealthConfig struct {
	CheckMinHealth     bool          `yaml:"checkMinHealth"`
	MinHealthThreshold float64       `yaml:"minHealthThreshold"`
	HealthCheckTimeout time.Duration `yaml:"healthCheckTimeout"`
}

type ElasticsearchIndexerConfig struct {
	Workers ElasticsearchWorkersConfig `yaml:"workers"`
	Retry   ElasticsearchRetryConfig   `yaml:"retry"`
}

type ElasticsearchWorkersConfig struct {
	Count         int           `yaml:"count"`
	BatchSize     int           `yaml:"batchSize"`
	FetchInterval time.Duration `yaml:"fetchInterval"`
}

type ElasticsearchRetryConfig struct {
	MaxRetries        int           `yaml:"maxRetries"`
	Delay             time.Duration `yaml:"delay"`
	ProcessingTimeout time.Duration `yaml:"processingTimeout"`
}

type SearchConfig struct {
	AllowedFields []string `yaml:"allowedFields"`
}

type AuthConfig struct {
	Email    string `yaml:"email"`
	Password string `yaml:"password"`
}
