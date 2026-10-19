package meili

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/meilisearch/meilisearch-go"
)

// Adapter wraps the official github.com/meilisearch/meilisearch-go SDK for search, health checks, and batch indexing.
type Adapter struct {
	client    meilisearch.ServiceManager
	index     meilisearch.IndexManager
	baseURL   string
	masterKey string
	indexUID  string
}

// NewAdapter constructs a new Meilisearch Adapter.
func NewAdapter(host, apiKey, indexUID string, customHTTP *http.Client) *Adapter {
	cleanHost := strings.TrimRight(strings.TrimSpace(host), "/")
	cleanKey := strings.TrimSpace(apiKey)
	if indexUID == "" {
		indexUID = "default_index"
	}

	var opts []meilisearch.Option
	if cleanKey != "" {
		opts = append(opts, meilisearch.WithAPIKey(cleanKey))
	}
	if customHTTP != nil {
		opts = append(opts, meilisearch.WithCustomClient(customHTTP))
	}

	var client meilisearch.ServiceManager
	var index meilisearch.IndexManager
	if cleanHost != "" {
		client = meilisearch.New(cleanHost, opts...)
		index = client.Index(indexUID)
	}

	return &Adapter{
		client:    client,
		index:     index,
		baseURL:   cleanHost,
		masterKey: cleanKey,
		indexUID:  indexUID,
	}
}

// Client returns the underlying meilisearch.ServiceManager.
func (a *Adapter) Client() meilisearch.ServiceManager {
	if a == nil {
		return nil
	}
	return a.client
}

// Index returns the underlying meilisearch.IndexManager.
func (a *Adapter) Index() meilisearch.IndexManager {
	if a == nil {
		return nil
	}
	return a.index
}

// SetHTTPClient overrides the HTTP client (useful for unit tests with httptest.Server).
func (a *Adapter) SetHTTPClient(client *http.Client) {
	if a != nil && client != nil && a.baseURL != "" {
		var opts []meilisearch.Option
		if a.masterKey != "" {
			opts = append(opts, meilisearch.WithAPIKey(a.masterKey))
		}
		opts = append(opts, meilisearch.WithCustomClient(client))
		a.client = meilisearch.New(a.baseURL, opts...)
		a.index = a.client.Index(a.indexUID)
	}
}

// IsEnabled returns true if Meilisearch URL and client are configured.
func (a *Adapter) IsEnabled() bool {
	return a != nil && a.client != nil && a.baseURL != ""
}

// UpdateSettingsAndWait updates the index settings and waits up to 10s for task completion.
func (a *Adapter) UpdateSettingsAndWait(ctx context.Context, settings *meilisearch.Settings) error {
	if !a.IsEnabled() || settings == nil {
		return nil
	}
	task, err := a.index.UpdateSettingsWithContext(ctx, settings)
	if err != nil {
		return fmt.Errorf("meilisearch update settings failed: %w", err)
	}
	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if _, err := a.client.WaitForTaskWithContext(waitCtx, task.TaskUID, 50*time.Millisecond); err != nil {
		return fmt.Errorf("meilisearch wait for settings task %d failed: %w", task.TaskUID, err)
	}
	return nil
}

// AddDocumentsAndWait pushes a batch of documents to Meilisearch and waits up to 10s for task completion.
func (a *Adapter) AddDocumentsAndWait(ctx context.Context, docs any, count int, primaryKey string) error {
	if !a.IsEnabled() || count == 0 {
		return nil
	}
	if primaryKey == "" {
		primaryKey = "id"
	}
	task, err := a.index.AddDocumentsWithContext(ctx, docs, &meilisearch.DocumentOptions{
		PrimaryKey: &primaryKey,
	})
	if err != nil {
		return fmt.Errorf("meilisearch add documents failed (count=%d): %w", count, err)
	}
	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if _, err := a.client.WaitForTaskWithContext(waitCtx, task.TaskUID, 50*time.Millisecond); err != nil {
		return fmt.Errorf("meilisearch wait for add documents task %d failed: %w", task.TaskUID, err)
	}
	return nil
}

// DeleteDocumentsAndWait removes document IDs from Meilisearch and waits up to 10s for task completion.
func (a *Adapter) DeleteDocumentsAndWait(ctx context.Context, ids []string) error {
	if !a.IsEnabled() || len(ids) == 0 {
		return nil
	}
	task, err := a.index.DeleteDocumentsWithContext(ctx, ids, nil)
	if err != nil {
		return fmt.Errorf("meilisearch delete documents failed (count=%d): %w", len(ids), err)
	}
	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if _, err := a.client.WaitForTaskWithContext(waitCtx, task.TaskUID, 50*time.Millisecond); err != nil {
		return fmt.Errorf("meilisearch wait for delete documents task %d failed: %w", task.TaskUID, err)
	}
	return nil
}

// Search executes a search query with limit, offset, and optional facets.
func (a *Adapter) Search(ctx context.Context, query string, limit, offset int64, facets ...string) (*meilisearch.SearchResponse, error) {
	if !a.IsEnabled() || a.index == nil {
		return nil, nil
	}
	ctxTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	return a.index.SearchWithContext(ctxTimeout, query, &meilisearch.SearchRequest{
		Limit:  limit,
		Offset: offset,
		Facets: facets,
	})
}

// IsHealthy checks if the Meilisearch instance is reachable and available.
func (a *Adapter) IsHealthy(ctx context.Context) bool {
	if !a.IsEnabled() {
		return false
	}
	ctxTimeout, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	health, err := a.client.HealthWithContext(ctxTimeout)
	return err == nil && health != nil && health.Status == "available"
}
