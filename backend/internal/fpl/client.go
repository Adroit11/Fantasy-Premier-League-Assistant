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
	requestTimeout = 10 * time.Second
	maxRPS         = 5
)

// Client interface abstracts FPL API communication for production and mock testing
type Client interface {
	GetBootstrapStatic(ctx context.Context) (*BootstrapStatic, error)
	GetEntry(ctx context.Context, teamID int) (*Entry, error)
	GetPicks(ctx context.Context, teamID int, event int) (*PicksResponse, error)
	GetFixtures(ctx context.Context, event *int) ([]Fixture, error)
	GetEntryHistory(ctx context.Context, teamID int) (*EntryHistoryResponse, error)
}

// HTTPClient implements Client using standard net/http and caching
type HTTPClient struct {
	baseURL    string
	userAgent  string
	httpClient *http.Client
	cache      Cache
	limiter    *tokenBucket
	group      flightGroup
}

func NewHTTPClient(cache Cache, baseURL string, userAgent string) *HTTPClient {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if userAgent == "" {
		userAgent = UserAgent
	}

	return &HTTPClient{
		baseURL:   baseURL,
		userAgent: userAgent,
		httpClient: &http.Client{
			Timeout: 12 * time.Second,
		},
		cache:   cache,
		limiter: newTokenBucket(maxRPS),
	}
}

func withRequestTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, requestTimeout)
}

func cacheGetTyped[T any](cache Cache, key string) (*T, bool) {
	if cache == nil {
		return nil, false
	}
	val, ok := cache.Get(key)
	if !ok {
		return nil, false
	}
	typed, ok := val.(*T)
	if !ok {
		return nil, false
	}
	return typed, true
}

func (c *HTTPClient) doRequest(ctx context.Context, endpoint string, target interface{}) error {
	ctx, cancel := withRequestTimeout(ctx)
	defer cancel()

	if err := c.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiter: %w", err)
	}

	reqURL := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
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
	if data, ok := cacheGetTyped[BootstrapStatic](c.cache, cacheKey); ok {
		return data, nil
	}

	v, err := c.group.Do(cacheKey, func() (interface{}, error) {
		if data, ok := cacheGetTyped[BootstrapStatic](c.cache, cacheKey); ok {
			return data, nil
		}
		var data BootstrapStatic
		if err := c.doRequest(ctx, "/bootstrap-static/", &data); err != nil {
			return nil, fmt.Errorf("GetBootstrapStatic: %w", err)
		}
		c.cache.Set(cacheKey, &data, 1*time.Hour)
		return &data, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*BootstrapStatic), nil
}

func (c *HTTPClient) GetEntry(ctx context.Context, teamID int) (*Entry, error) {
	cacheKey := fmt.Sprintf("fpl:entry:%d", teamID)
	if data, ok := cacheGetTyped[Entry](c.cache, cacheKey); ok {
		return data, nil
	}

	v, err := c.group.Do(cacheKey, func() (interface{}, error) {
		if data, ok := cacheGetTyped[Entry](c.cache, cacheKey); ok {
			return data, nil
		}
		var data Entry
		if err := c.doRequest(ctx, fmt.Sprintf("/entry/%d/", teamID), &data); err != nil {
			return nil, fmt.Errorf("GetEntry: %w", err)
		}
		c.cache.Set(cacheKey, &data, 15*time.Minute)
		return &data, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*Entry), nil
}

func (c *HTTPClient) GetPicks(ctx context.Context, teamID int, event int) (*PicksResponse, error) {
	cacheKey := fmt.Sprintf("fpl:picks:%d:gw:%d", teamID, event)
	if data, ok := cacheGetTyped[PicksResponse](c.cache, cacheKey); ok {
		return data, nil
	}

	v, err := c.group.Do(cacheKey, func() (interface{}, error) {
		if data, ok := cacheGetTyped[PicksResponse](c.cache, cacheKey); ok {
			return data, nil
		}
		var data PicksResponse
		if err := c.doRequest(ctx, fmt.Sprintf("/entry/%d/event/%d/picks/", teamID, event), &data); err != nil {
			return nil, fmt.Errorf("GetPicks: %w", err)
		}
		c.cache.Set(cacheKey, &data, 5*time.Minute)
		return &data, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*PicksResponse), nil
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

	v, err := c.group.Do(cacheKey, func() (interface{}, error) {
		if val, ok := c.cache.Get(cacheKey); ok {
			if data, ok := val.([]Fixture); ok {
				return data, nil
			}
		}
		var data []Fixture
		if err := c.doRequest(ctx, endpoint, &data); err != nil {
			return nil, fmt.Errorf("GetFixtures: %w", err)
		}
		c.cache.Set(cacheKey, data, 3*time.Hour)
		return data, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]Fixture), nil
}

// GetEntryHistory fetches per-gameweek points history from /api/entry/{id}/history/.
// Results are cached for 5 minutes since they only change after a GW closes.
func (c *HTTPClient) GetEntryHistory(ctx context.Context, teamID int) (*EntryHistoryResponse, error) {
	cacheKey := fmt.Sprintf("fpl:entry_history:%d", teamID)
	if data, ok := cacheGetTyped[EntryHistoryResponse](c.cache, cacheKey); ok {
		return data, nil
	}

	v, err := c.group.Do(cacheKey, func() (interface{}, error) {
		if data, ok := cacheGetTyped[EntryHistoryResponse](c.cache, cacheKey); ok {
			return data, nil
		}
		var data EntryHistoryResponse
		if err := c.doRequest(ctx, fmt.Sprintf("/entry/%d/history/", teamID), &data); err != nil {
			return nil, fmt.Errorf("GetEntryHistory: %w", err)
		}
		c.cache.Set(cacheKey, &data, 5*time.Minute)
		return &data, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*EntryHistoryResponse), nil
}
