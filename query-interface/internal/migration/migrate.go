package migration

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

//go:embed mapping.json
var mappingJSON string

func Migrate(ctx context.Context, client *elasticsearch.Client, indexName string) error {
	exists, err := indexExists(ctx, client, indexName)
	if err != nil {
		return fmt.Errorf("failed to check if index exists: %w", err)
	}

	if exists {
		return nil // already exists
	}

	err = createIndex(ctx, client, indexName, mappingJSON)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	return nil
}

func indexExists(ctx context.Context, client *elasticsearch.Client, indexName string) (bool, error) {
	req := esapi.IndicesExistsRequest{
		Index: []string{indexName},
	}

	res, err := req.Do(ctx, client)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	return res.StatusCode == 200, nil
}

func createIndex(ctx context.Context, client *elasticsearch.Client, indexName, mapping string) error {
	req := esapi.IndicesCreateRequest{
		Index: indexName,
		Body:  strings.NewReader(mapping),
	}

	res, err := req.Do(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("elasticsearch error: %s", string(body))
	}

	return nil
}

func MigrateWithSettings(ctx context.Context, client *elasticsearch.Client, indexName string, customMapping string) error {
	exists, err := indexExists(ctx, client, indexName)
	if err != nil {
		return fmt.Errorf("failed to check if index exists: %w", err)
	}

	if exists {
		return nil // already exists
	}

	mapping := mappingJSON
	if customMapping != "" {
		mapping = customMapping // use custom mapping
	}

	err = createIndex(ctx, client, indexName, mapping)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	return nil
}
