package config

import (
	"time"
)

// application configuration structure
type Config struct {
	Server        ServerConfig        `yaml:"server"`
	Postgres      PostgresConfig      `yaml:"postgres"`
	Elasticsearch ElasticsearchConfig `yaml:"elasticsearch"`
	Indexer       IndexerConfig       `yaml:"indexer"`
	Filters       FiltersConfig       `yaml:"filters"`
}

// server configuration structure
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// postgres database configuration structure
type PostgresConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	DBName          string        `yaml:"dbname"`
	MaxOpenConns    int           `yaml:"maxOpenConns"`
	MaxIdleConns    int           `yaml:"maxIdleConns"`
	ConnMaxLifetime time.Duration `yaml:"connMaxLifetime"`
}

// elasticsearch configuration structure
type ElasticsearchConfig struct {
	Host       string        `yaml:"host"`
	Index      string        `yaml:"index"`
	BatchSize  int           `yaml:"batchSize"`
	RetryCount int           `yaml:"retryCount"`
	Timeout    time.Duration `yaml:"timeout"`
}

// indexer configuration structure
type IndexerConfig struct {
	WorkerCount       int           `yaml:"workerCount"`
	FetchInterval     time.Duration `yaml:"fetchInterval"`
	BatchSize         int           `yaml:"batchSize"`
	MaxRetries        int           `yaml:"maxRetries"`
	ProcessingTimeout time.Duration `yaml:"processingTimeout"`
}

// filters configuration structure
type FiltersConfig struct {
	AllowedFields []string `yaml:"allowedFields"`
}
