package config

import (
	"time"
)

type Config struct {
	Auth        SharedAuthConfig  `yaml:"auth"`
	Database    DatabaseConfig    `yaml:"database"`
	DLQ         DLQConfig         `yaml:"dlq"`
	LogIngestor LogIngestorConfig `yaml:"log-ingestor"`
}

// SharedAuthConfig holds the jwt secret shared with query-interface. log-ingestor
// only validates tokens against it; query-interface is the one that issues them.
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

type LogIngestorConfig struct {
	Server     ServerConfig         `yaml:"server"`
	Processing ProcessingConfig     `yaml:"processing"`
	DLQ        LogIngestorDLQConfig `yaml:"dlq"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type ProcessingConfig struct {
	Batcher BatcherConfig `yaml:"batcher"`
	Workers WorkersConfig `yaml:"workers"`
}

type BatcherConfig struct {
	MaxBatchSize  int           `yaml:"maxBatchSize"`
	MaxBatchCount int           `yaml:"maxBatchCount"`
	FlushInterval time.Duration `yaml:"flushInterval"`
}

type WorkersConfig struct {
	Concurrency   int           `yaml:"concurrency"`
	RetryInterval time.Duration `yaml:"retryInterval"`
	RetryCount    int           `yaml:"retryCount"`
}

type LogIngestorDLQConfig struct {
	FailureTypes LogIngestorFailureTypesConfig `yaml:"failureTypes"`
}

type LogIngestorFailureTypesConfig struct {
	ParseError      bool `yaml:"parseError"`
	DBFailure       bool `yaml:"dbFailure"`
	ValidationError bool `yaml:"validationError"`
	TimeoutError    bool `yaml:"timeoutError"`
}
