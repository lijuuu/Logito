package indexer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/lijuuu/Logito/query-interface/internal/migration"
	"github.com/lijuuu/Logito/query-interface/pkg/logentry"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

type ESClient struct {
	client       *elasticsearch.Client
	indexName    string
	timeout      time.Duration
	healthConfig ElasticsearchHealthConfig
	poolConfig   ElasticsearchPoolConfig
}

type ElasticsearchPoolConfig struct {
	MaxConnsPerHost int
	MaxIdleConns    int
	IdleConnTimeout time.Duration
}

type ElasticsearchHealthConfig struct {
	CheckMinHealth     bool
	MinHealthThreshold float64
	HealthCheckTimeout time.Duration
}

type ESHealthStatus struct {
	Status      string  `json:"status"`
	HealthScore float64 `json:"health_score"`
	Healthy     bool    `json:"healthy"`
}

func NewESClient(host, indexName string, timeout time.Duration, poolConfig ElasticsearchPoolConfig, healthConfig ElasticsearchHealthConfig) (*ESClient, error) {
	transport := &http.Transport{
		MaxIdleConns:    poolConfig.MaxIdleConns,
		MaxConnsPerHost: poolConfig.MaxConnsPerHost,
		IdleConnTimeout: poolConfig.IdleConnTimeout,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	cfg := elasticsearch.Config{
		Addresses:     []string{host},
		RetryOnStatus: []int{502, 503, 504, 429},
		MaxRetries:    3,
		RetryBackoff: func(i int) time.Duration {
			return time.Duration(i) * time.Second
		},
		Transport: transport,
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create es client: %w", err)
	}

	esClient := &ESClient{
		client:       client,
		indexName:    indexName,
		timeout:      timeout,
		healthConfig: healthConfig,
		poolConfig:   poolConfig,
	}

	if err := esClient.HealthCheck(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to connect to elasticsearch: %w", err)
	}

	return esClient, nil
}

func (c *ESClient) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	res, err := c.client.Cluster.Health(
		c.client.Cluster.Health.WithContext(ctx),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("elasticsearch health check failed: %s", res.String())
	}

	return nil
}

// GetHealthStatus returns detailed health information including health score
func (c *ESClient) GetHealthStatus(ctx context.Context) (*ESHealthStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, c.healthConfig.HealthCheckTimeout)
	defer cancel()

	res, err := c.client.Cluster.Health(
		c.client.Cluster.Health.WithContext(ctx),
		c.client.Cluster.Health.WithLevel("indices"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster health: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("elasticsearch health check failed: %s", res.String())
	}

	var healthResponse struct {
		Status                  string `json:"status"`
		ActiveShards            int    `json:"active_shards"`
		RelocatingShards        int    `json:"relocating_shards"`
		InitializingShards      int    `json:"initializing_shards"`
		UnassignedShards        int    `json:"unassigned_shards"`
		DelayedUnassignedShards int    `json:"delayed_unassigned_shards"`
		NumberOfNodes           int    `json:"number_of_nodes"`
		NumberOfDataNodes       int    `json:"number_of_data_nodes"`
		ActivePrimaryShards     int    `json:"active_primary_shards"`
	}

	if err := json.NewDecoder(res.Body).Decode(&healthResponse); err != nil {
		return nil, fmt.Errorf("failed to decode health response: %w", err)
	}

	// Calculate health score based on shard status
	totalShards := healthResponse.ActiveShards + healthResponse.RelocatingShards +
		healthResponse.InitializingShards + healthResponse.UnassignedShards +
		healthResponse.DelayedUnassignedShards

	var healthScore float64
	if totalShards > 0 {
		healthyShards := healthResponse.ActiveShards + healthResponse.RelocatingShards
		healthScore = float64(healthyShards) / float64(totalShards) * 100
	} else {
		healthScore = 100.0 // No shards means healthy
	}

	// Determine if cluster is healthy based on status and score
	healthy := healthResponse.Status == "green" ||
		(healthResponse.Status == "yellow" && healthScore >= c.healthConfig.MinHealthThreshold)

	return &ESHealthStatus{
		Status:      healthResponse.Status,
		HealthScore: healthScore,
		Healthy:     healthy,
	}, nil
}

// IsHealthyForIndexing checks if ES is healthy enough for indexing
func (c *ESClient) IsHealthyForIndexing(ctx context.Context) (bool, error) {
	if !c.healthConfig.CheckMinHealth {
		return true, nil 
	}

	healthStatus, err := c.GetHealthStatus(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check ES health: %w", err)
	}

	return healthStatus.Healthy && healthStatus.HealthScore >= c.healthConfig.MinHealthThreshold, nil
}

// GetPoolStats returns connection pool statistics
func (c *ESClient) GetPoolStats() map[string]interface{} {
	return map[string]interface{}{
		"maxIdleConns":    c.poolConfig.MaxIdleConns,
		"maxConnsPerHost": c.poolConfig.MaxConnsPerHost,
		"idleConnTimeout": c.poolConfig.IdleConnTimeout,
	}
}

func (c *ESClient) CreateIndex(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	err := migration.Migrate(ctx, c.client, c.indexName)
	if err != nil {
		return fmt.Errorf("failed to run elasticsearch migration: %w", err)
	}

	return nil
}

func (c *ESClient) DeleteIndex(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req := esapi.IndicesDeleteRequest{
		Index: []string{c.indexName},
	}

	res, err := req.Do(ctx, c.client)
	if err != nil {
		return fmt.Errorf("failed to delete index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("elasticsearch delete index error: %s", string(body))
	}

	return nil
}

func (c *ESClient) CreateIndexWithCustomMapping(ctx context.Context, customMapping string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	err := migration.MigrateWithSettings(ctx, c.client, c.indexName, customMapping)
	if err != nil {
		return fmt.Errorf("failed to run elasticsearch migration with custom mapping: %w", err)
	}

	return nil
}

func (c *ESClient) IndexExists(ctx context.Context) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req := esapi.IndicesExistsRequest{
		Index: []string{c.indexName},
	}

	res, err := req.Do(ctx, c.client)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	return res.StatusCode == 200, nil
}

func (c *ESClient) BulkIndex(ctx context.Context, entries []*logentry.LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var buf bytes.Buffer
	for _, entry := range entries {
		action := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": c.indexName,
				"_id":    fmt.Sprintf("%d", entry.ID),
			},
		}

		actionJSON, err := json.Marshal(action)
		if err != nil {
			return fmt.Errorf("failed to marshal action: %w", err)
		}
		buf.Write(actionJSON)
		buf.WriteByte('\n')

		docJSON, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal document: %w", err)
		}
		buf.Write(docJSON)
		buf.WriteByte('\n')
	}

	req := esapi.BulkRequest{
		Body:    &buf,
		Refresh: "false",
	}

	res, err := req.Do(ctx, c.client)
	if err != nil {
		return fmt.Errorf("failed to execute bulk request: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("elasticsearch bulk error: %s", string(body))
	}

	return nil
}

func (c *ESClient) Search(ctx context.Context, query map[string]interface{}) (*SearchResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	req := esapi.SearchRequest{
		Index: []string{c.indexName},
		Body:  strings.NewReader(string(queryJSON)),
	}

	res, err := req.Do(ctx, c.client)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("elasticsearch search error: %s", string(body))
	}

	var searchResp SearchResponse
	if err := json.NewDecoder(res.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	return &searchResp, nil
}

type SearchResponse struct {
	Took         int64                  `json:"took"`
	TimedOut     bool                   `json:"timed_out"`
	Aggregations map[string]interface{} `json:"aggregations"`
	Hits         struct {
		Total struct {
			Value int64 `json:"value"`
		} `json:"total"`
		Hits []struct {
			Index  string                 `json:"_index"`
			ID     string                 `json:"_id"`
			Score  float64                `json:"_score"`
			Source map[string]interface{} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}
