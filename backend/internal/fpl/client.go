package fpl

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	DefaultBaseURL = "https://fantasy.premierleague.com/api"
	UserAgent      = "FPL-Assistant-Desktop/1.0 (Go; Windows)"
)

// Client interface abstracts FPL API communication for production and mock testing
type Client interface {
	GetBootstrapStatic(ctx context.Context) (*BootstrapStatic, error)
	GetEntry(ctx context.Context, teamID int) (*Entry, error)
	GetPicks(ctx context.Context, teamID int, event int) (*PicksResponse, error)
	GetFixtures(ctx context.Context, event *int) ([]Fixture, error)
}

// HTTPClient implements Client using standard net/http and caching
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
	cache      Cache
}

func NewHTTPClient(cache Cache) *HTTPClient {
	return &HTTPClient{
		baseURL: DefaultBaseURL,
		httpClient: &http.Client{
			Timeout: 12 * time.Second,
		},
		cache: cache,
	}
}

func (c *HTTPClient) doRequest(ctx context.Context, endpoint string, target interface{}) error {
	reqURL := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("fpl api rate limited (HTTP 429)")
	}

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("resource not found (HTTP 404)")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("fpl api error: status code %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("failed to decode response json: %w", err)
	}

	return nil
}

func (c *HTTPClient) GetBootstrapStatic(ctx context.Context) (*BootstrapStatic, error) {
	cacheKey := "fpl:bootstrap_static"
	if val, ok := c.cache.Get(cacheKey); ok {
		if data, ok := val.(*BootstrapStatic); ok {
			return data, nil
		}
	}

	var data BootstrapStatic
	if err := c.doRequest(ctx, "/bootstrap-static/", &data); err != nil {
		return nil, fmt.Errorf("GetBootstrapStatic: %w", err)
	}

	// Cache for 1 hour
	c.cache.Set(cacheKey, &data, 1*time.Hour)
	return &data, nil
}

func (c *HTTPClient) GetEntry(ctx context.Context, teamID int) (*Entry, error) {
	cacheKey := fmt.Sprintf("fpl:entry:%d", teamID)
	if val, ok := c.cache.Get(cacheKey); ok {
		if data, ok := val.(*Entry); ok {
			return data, nil
		}
	}

	var data Entry
	if err := c.doRequest(ctx, fmt.Sprintf("/entry/%d/", teamID), &data); err != nil {
		return nil, fmt.Errorf("GetEntry: %w", err)
	}

	// Cache for 15 minutes
	c.cache.Set(cacheKey, &data, 15*time.Minute)
	return &data, nil
}

func (c *HTTPClient) GetPicks(ctx context.Context, teamID int, event int) (*PicksResponse, error) {
	cacheKey := fmt.Sprintf("fpl:picks:%d:gw:%d", teamID, event)
	if val, ok := c.cache.Get(cacheKey); ok {
		if data, ok := val.(*PicksResponse); ok {
			return data, nil
		}
	}

	var data PicksResponse
	if err := c.doRequest(ctx, fmt.Sprintf("/entry/%d/event/%d/picks/", teamID, event), &data); err != nil {
		return nil, fmt.Errorf("GetPicks: %w", err)
	}

	// Cache for 5 minutes
	c.cache.Set(cacheKey, &data, 5*time.Minute)
	return &data, nil
}

func (c *HTTPClient) GetFixtures(ctx context.Context, event *int) ([]Fixture, error) {
	cacheKey := "fpl:fixtures:all"
	endpoint := "/fixtures/"
	if event != nil {
		cacheKey = fmt.Sprintf("fpl:fixtures:gw:%d", *event)
		endpoint = fmt.Sprintf("/fixtures/?event=%d", *event)
	}

	if val, ok := c.cache.Get(cacheKey); ok {
		if data, ok := val.([]Fixture); ok {
			return data, nil
		}
	}

	var data []Fixture
	if err := c.doRequest(ctx, endpoint, &data); err != nil {
		return nil, fmt.Errorf("GetFixtures: %w", err)
	}

	// Cache for 2 hours
	c.cache.Set(cacheKey, data, 2*time.Hour)
	return data, nil
}
