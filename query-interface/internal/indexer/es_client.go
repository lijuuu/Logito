package indexer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/lijuuu/Logito/query-interface/internal/migration"
	"github.com/lijuuu/Logito/query-interface/pkg/logentry"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

type ESClient struct {
	client    *elasticsearch.Client
	indexName string
	timeout   time.Duration
}

func NewESClient(host, indexName string, timeout time.Duration) (*ESClient, error) {
	cfg := elasticsearch.Config{
		Addresses:     []string{host},
		RetryOnStatus: []int{502, 503, 504, 429}, // retry config
		MaxRetries:    3,
		RetryBackoff: func(i int) time.Duration {
			return time.Duration(i) * time.Second
		},
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create es client: %w", err)
	}

	esClient := &ESClient{
		client:    client,
		indexName: indexName,
		timeout:   timeout,
	}

	// test connection
	if err := esClient.HealthCheck(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to connect to elasticsearch: %w", err)
	}

	return esClient, nil
}

// healthcheck for elasticsearch cluster
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

// creates the logs index with proper mapping using migration
func (c *ESClient) CreateIndex(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	err := migration.Migrate(ctx, c.client, c.indexName)
	if err != nil {
		return fmt.Errorf("failed to run elasticsearch migration: %w", err)
	}

	return nil
}

// creates the logs index with custom mapping using migration
func (c *ESClient) CreateIndexWithCustomMapping(ctx context.Context, customMapping string) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	err := migration.MigrateWithSettings(ctx, c.client, c.indexName, customMapping)
	if err != nil {
		return fmt.Errorf("failed to run elasticsearch migration with custom mapping: %w", err)
	}

	return nil
}

// checks if the index exists
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

// indexes multiple log entries using bulk api
func (c *ESClient) BulkIndex(ctx context.Context, entries []*logentry.LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// prepare bulk request body
	var buf bytes.Buffer
	for _, entry := range entries {
		// create index action
		action := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": c.indexName,
				"_id":    fmt.Sprintf("%d", entry.ID),
			},
		}

		// serialize action
		actionJSON, err := json.Marshal(action)
		if err != nil {
			return fmt.Errorf("failed to marshal action: %w", err)
		}
		buf.Write(actionJSON)
		buf.WriteByte('\n')

		// serialize document
		docJSON, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal document: %w", err)
		}
		buf.Write(docJSON)
		buf.WriteByte('\n')
	}

	// execute bulk request
	req := esapi.BulkRequest{
		Body:    &buf,
		Refresh: "false", // don't refresh for better performance
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

// performs a search query on the index
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

// elasticsearch search response structure
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
