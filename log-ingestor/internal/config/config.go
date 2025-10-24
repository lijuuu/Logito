package config

import (
	"time"
)

//Config represents the application configuration
type Config struct {
	Server        ServerConfig        `yaml:"server"`
	Postgres      PostgresConfig      `yaml:"postgres"`
	Elasticsearch ElasticsearchConfig `yaml:"elasticsearch"`
	Batcher       BatcherConfig       `yaml:"batcher"`
	Worker        WorkerConfig        `yaml:"worker"`
}

//ServerConfig represents server configuration
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

//PostgresConfig represents postgres database configuration
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

//ElasticsearchConfig represents elasticsearch configuration
type ElasticsearchConfig struct {
	Host       string        `yaml:"host"`
	Index      string        `yaml:"index"`
	BatchSize  int           `yaml:"batchSize"`
	RetryCount int           `yaml:"retryCount"`
	Timeout    time.Duration `yaml:"timeout"`
}

//BatcherConfig represents batcher configuration
type BatcherConfig struct {
	MaxBatchSize  int           `yaml:"maxBatchSize"`
	MaxBatchCount int           `yaml:"maxBatchCount"`
	FlushInterval time.Duration `yaml:"flushInterval"`
}

//WorkerConfig represents worker configuration
type WorkerConfig struct {
	Concurrency   int           `yaml:"concurrency"`
	RetryInterval time.Duration `yaml:"retryInterval"`
	WorkerSize    int           `yaml:"workerSize"`
}
