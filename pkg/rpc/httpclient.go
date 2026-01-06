package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
)

// HTTPClient is a wrapper around an http.Client with circuit-breaker and token-bucket rate limiting.
type HTTPClient struct {
	endpoints []string
	client    *http.Client

	// token-bucket
	tokens      int64
	maxTokens   int64
	refillEvery time.Duration
	lastRefill  atomic.Value // time.Time

	// circuit-breaker
	mu       sync.Mutex
	failures map[string]int
	opened   map[string]time.Time

	breakerThreshold int
	breakerCooldown  time.Duration

	// response cache for H-1 data (avoids refetching previous height)
	cacheMu     sync.RWMutex
	cache       map[string]any // key: "path:height"
	cacheHeight uint64         // current cached height (only keep last 2 heights)
}

// Opts is the set of options for a new HTTPClient.
type Opts struct {
	Endpoints       []string
	Timeout         time.Duration
	RPS             int
	Burst           int
	BreakerFailures int
	BreakerCooldown time.Duration
	HTTPClient      *http.Client
}

// NewHTTPWithOpts creates a new HTTPClient with the given options.
func NewHTTPWithOpts(o Opts) *HTTPClient {
	if o.RPS <= 0 {
		o.RPS = 20
	}
	if o.Burst <= 0 {
		o.Burst = 40
	}
	if o.Timeout <= 0 {
		o.Timeout = 15 * time.Second
	}
	if o.BreakerFailures <= 0 {
		o.BreakerFailures = 3
	}
	if o.BreakerCooldown <= 0 {
		o.BreakerCooldown = 5 * time.Second
	}

	client := o.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: o.Timeout}
	}

	c := &HTTPClient{
		endpoints:        dedup(o.Endpoints),
		client:           client,
		maxTokens:        int64(o.Burst),
		refillEvery:      time.Second / time.Duration(o.RPS),
		failures:         map[string]int{},
		opened:           map[string]time.Time{},
		breakerThreshold: o.BreakerFailures,
		breakerCooldown:  o.BreakerCooldown,
		cache:            make(map[string]any),
	}
	c.tokens = c.maxTokens
	c.lastRefill.Store(time.Now())
	return c
}

func dedup(ss []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(ss))
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// cacheKey returns the cache key for a path and height.
func cacheKey(path string, height uint64) string {
	return fmt.Sprintf("%s:%d", path, height)
}

// getCache returns cached data if available.
func getCache[T any](c *HTTPClient, path string, height uint64) (T, bool) {
	var zero T
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()

	key := cacheKey(path, height)
	if v, ok := c.cache[key]; ok {
		if typed, ok := v.(T); ok {
			slog.Debug("rpc cache hit", "path", path, "height", height)
			return typed, true
		}
	}
	return zero, false
}

// setCache stores data in the cache, retaining the last 10 blocks.
func setCache[T any](c *HTTPClient, path string, height uint64, data T) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()

	// Track this height
	if height > c.cacheHeight {
		c.cacheHeight = height
	}

	// Clean entries older than 10 blocks from the highest seen
	minHeight := uint64(0)
	if c.cacheHeight > 10 {
		minHeight = c.cacheHeight - 10
	}

	for k := range c.cache {
		// Extract height from key "path:height"
		var h uint64
		// Find the last colon and parse the height after it
		for i := len(k) - 1; i >= 0; i-- {
			if k[i] == ':' {
				fmt.Sscanf(k[i+1:], "%d", &h)
				break
			}
		}
		if h > 0 && h < minHeight {
			delete(c.cache, k)
		}
	}

	key := cacheKey(path, height)
	c.cache[key] = data
}

func (c *HTTPClient) refill() {
	last := c.lastRefill.Load().(time.Time)
	now := time.Now()
	if now.Sub(last) >= c.refillEvery {
		if atomic.LoadInt64(&c.tokens) < c.maxTokens {
			atomic.AddInt64(&c.tokens, 1)
		}
		c.lastRefill.Store(now)
	}
}

func (c *HTTPClient) acquire() {
	for {
		c.refill()
		if atomic.LoadInt64(&c.tokens) > 0 {
			atomic.AddInt64(&c.tokens, -1)
			return
		}
		time.Sleep(c.refillEvery / 2)
	}
}

func (c *HTTPClient) isOpen(ep string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	until, ok := c.opened[ep]
	if !ok {
		return false
	}
	if time.Now().After(until) {
		delete(c.opened, ep)
		c.failures[ep] = 0
		return false
	}
	return true
}

func (c *HTTPClient) noteFailure(ep string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures[ep]++
	if c.failures[ep] >= c.breakerThreshold {
		c.opened[ep] = time.Now().Add(c.breakerCooldown)
	}
}

func (c *HTTPClient) doJSON(ctx context.Context, method, path string, payload any, out any) error {
	if len(c.endpoints) == 0 {
		return fmt.Errorf("no endpoints configured")
	}

	var lastErr error
	for i := 0; i < len(c.endpoints); i++ {
		ep := c.endpoints[i%len(c.endpoints)]
		if c.isOpen(ep) {
			continue
		}

		c.acquire()

		var body *bytes.Reader
		if payload != nil {
			b, mErr := json.Marshal(payload)
			if mErr != nil {
				return mErr
			}
			body = bytes.NewReader(b)
		} else {
			body = bytes.NewReader(nil)
		}

		req, reqErr := http.NewRequestWithContext(ctx, method, ep+path, body)
		if reqErr != nil {
			return reqErr
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			c.noteFailure(ep)
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server %d", resp.StatusCode)
			c.noteFailure(ep)
			resp.Body.Close()
			continue
		}
		if resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("http %d", resp.StatusCode)
			resp.Body.Close()
			continue
		}

		if out != nil {
			// Read raw body for debugging
			rawBody, err := io.ReadAll(resp.Body)
			if err != nil {
				resp.Body.Close()
				lastErr = err
				continue
			}

			slog.Debug("rpc", "path", path, "len", len(rawBody))

			if err := json.Unmarshal(rawBody, out); err != nil {
				resp.Body.Close()
				lastErr = fmt.Errorf("json unmarshal: %w (body: %s)", err, string(rawBody[:min(200, len(rawBody))]))
				continue
			}
		}

		resp.Body.Close()
		return nil
	}

	return lastErr
}

// QueryByHeightRequest is the request for height-based queries.
type QueryByHeightRequest struct {
	Height uint64 `json:"height"`
}

// ChainHead returns the current chain height.
func (c *HTTPClient) ChainHead(ctx context.Context) (uint64, error) {
	var resp struct {
		Height uint64 `json:"height"`
	}
	if err := c.doJSON(ctx, http.MethodGet, headPath, nil, &resp); err != nil {
		return 0, err
	}
	return resp.Height, nil
}

// BlockByHeight fetches a block at the given height.
func (c *HTTPClient) BlockByHeight(ctx context.Context, height uint64) (*lib.BlockResult, error) {
	// Pre-allocate nested pointer structs so json.Unmarshal can call their custom UnmarshalJSON
	resp := lib.BlockResult{
		BlockHeader: &lib.BlockHeader{},
		Meta:        &lib.BlockResultMeta{},
	}
	if err := c.doJSON(ctx, http.MethodPost, blockByHeightPath, QueryByHeightRequest{Height: height}, &resp); err != nil {
		return nil, err
	}
	// Check if block is ready (height should match request)
	if resp.BlockHeader.Height != height {
		return nil, fmt.Errorf("block not ready: requested height %d, got %d", height, resp.BlockHeader.Height)
	}
	return &resp, nil
}

// TxsByHeight fetches transactions at the given height.
func (c *HTTPClient) TxsByHeight(ctx context.Context, height uint64) ([]*lib.TxResult, error) {
	return listPaged[*lib.TxResult](ctx, c, txsByHeightPath, QueryByHeightRequest{Height: height})
}

// EventsByHeight fetches events at the given height.
func (c *HTTPClient) EventsByHeight(ctx context.Context, height uint64) ([]*lib.Event, error) {
	return listPaged[*lib.Event](ctx, c, eventsByHeightPath, QueryByHeightRequest{Height: height})
}

// AccountsByHeight fetches accounts at the given height.
func (c *HTTPClient) AccountsByHeight(ctx context.Context, height uint64) ([]*fsm.Account, error) {
	return listPaged[*fsm.Account](ctx, c, accountsByHeightPath, QueryByHeightRequest{Height: height})
}

// ValidatorsByHeight fetches validators at the given height.
func (c *HTTPClient) ValidatorsByHeight(ctx context.Context, height uint64) ([]*fsm.Validator, error) {
	if cached, ok := getCache[[]*fsm.Validator](c, validatorsPath, height); ok {
		return cached, nil
	}
	resp, err := listPaged[*fsm.Validator](ctx, c, validatorsPath, QueryByHeightRequest{Height: height})
	if err != nil {
		return nil, err
	}
	setCache(c, validatorsPath, height, resp)
	return resp, nil
}

// OrdersByHeight fetches orders at the given height.
// Note: This endpoint returns an array of OrderBook objects, each containing orders for a chain.
func (c *HTTPClient) OrdersByHeight(ctx context.Context, height uint64) ([]*lib.SellOrder, error) {
	var orderBooks []*lib.OrderBook
	if err := c.doJSON(ctx, http.MethodPost, ordersByHeightPath, QueryByHeightRequest{Height: height}, &orderBooks); err != nil {
		return nil, err
	}

	// Flatten all orders from all order books
	var allOrders []*lib.SellOrder
	for _, ob := range orderBooks {
		allOrders = append(allOrders, ob.Orders...)
	}
	return allOrders, nil
}

// DexPricesByHeight fetches DEX prices at the given height.
// Note: This endpoint returns a plain array, not a paginated response.
func (c *HTTPClient) DexPricesByHeight(ctx context.Context, height uint64) ([]*lib.DexPrice, error) {
	var resp []*lib.DexPrice
	if err := c.doJSON(ctx, http.MethodPost, dexPricePath, QueryByHeightRequest{Height: height}, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// PoolsByHeight fetches pools at the given height.
func (c *HTTPClient) PoolsByHeight(ctx context.Context, height uint64) ([]*fsm.Pool, error) {
	if cached, ok := getCache[[]*fsm.Pool](c, poolsPath, height); ok {
		return cached, nil
	}
	resp, err := listPaged[*fsm.Pool](ctx, c, poolsPath, QueryByHeightRequest{Height: height})
	if err != nil {
		return nil, err
	}
	setCache(c, poolsPath, height, resp)
	return resp, nil
}

// AllDexBatchesByHeight fetches current DEX batches at the given height.
// Note: This endpoint returns a plain array, not a paginated response.
func (c *HTTPClient) AllDexBatchesByHeight(ctx context.Context, height uint64) ([]*lib.DexBatch, error) {
	if cached, ok := getCache[[]*lib.DexBatch](c, dexBatchByHeightPath, height); ok {
		return cached, nil
	}
	var resp []*lib.DexBatch
	if err := c.doJSON(ctx, http.MethodPost, dexBatchByHeightPath, QueryByHeightRequest{Height: height}, &resp); err != nil {
		return nil, err
	}
	setCache(c, dexBatchByHeightPath, height, resp)
	return resp, nil
}

// AllNextDexBatchesByHeight fetches next DEX batches at the given height.
// Note: This endpoint returns a plain array, not a paginated response.
func (c *HTTPClient) AllNextDexBatchesByHeight(ctx context.Context, height uint64) ([]*lib.DexBatch, error) {
	if cached, ok := getCache[[]*lib.DexBatch](c, nextDexBatchByHeightPath, height); ok {
		return cached, nil
	}
	var resp []*lib.DexBatch
	if err := c.doJSON(ctx, http.MethodPost, nextDexBatchByHeightPath, QueryByHeightRequest{Height: height}, &resp); err != nil {
		return nil, err
	}
	setCache(c, nextDexBatchByHeightPath, height, resp)
	return resp, nil
}

// AllParamsByHeight fetches all params at the given height.
func (c *HTTPClient) AllParamsByHeight(ctx context.Context, height uint64) (*fsm.Params, error) {
	var resp fsm.Params
	if err := c.doJSON(ctx, http.MethodPost, allParamsPath, QueryByHeightRequest{Height: height}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ValParamsByHeight fetches validator params at the given height.
func (c *HTTPClient) ValParamsByHeight(ctx context.Context, height uint64) (*fsm.ValidatorParams, error) {
	var resp fsm.ValidatorParams
	if err := c.doJSON(ctx, http.MethodPost, valParamsPath, QueryByHeightRequest{Height: height}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// NonSignersByHeight fetches non-signers at the given height.
func (c *HTTPClient) NonSignersByHeight(ctx context.Context, height uint64) ([]*fsm.NonSigner, error) {
	if cached, ok := getCache[[]*fsm.NonSigner](c, nonSignersPath, height); ok {
		return cached, nil
	}
	resp, err := listPaged[*fsm.NonSigner](ctx, c, nonSignersPath, QueryByHeightRequest{Height: height})
	if err != nil {
		return nil, err
	}
	setCache(c, nonSignersPath, height, resp)
	return resp, nil
}

// DoubleSignersByHeight fetches double-signers at the given height.
func (c *HTTPClient) DoubleSignersByHeight(ctx context.Context, height uint64) ([]*lib.DoubleSigner, error) {
	if cached, ok := getCache[[]*lib.DoubleSigner](c, doubleSignersPath, height); ok {
		return cached, nil
	}
	resp, err := listPaged[*lib.DoubleSigner](ctx, c, doubleSignersPath, QueryByHeightRequest{Height: height})
	if err != nil {
		return nil, err
	}
	setCache(c, doubleSignersPath, height, resp)
	return resp, nil
}

// CommitteesDataByHeight fetches committees data at the given height.
// Note: This endpoint returns {"list": [...]} not a paginated response.
func (c *HTTPClient) CommitteesDataByHeight(ctx context.Context, height uint64) ([]*lib.CommitteeData, error) {
	var resp struct {
		List []*lib.CommitteeData `json:"list"`
	}
	if err := c.doJSON(ctx, http.MethodPost, committeesDataPath, QueryByHeightRequest{Height: height}, &resp); err != nil {
		return nil, err
	}
	return resp.List, nil
}

// SubsidizedCommitteesByHeight fetches subsidized committee IDs at the given height.
// Note: This endpoint returns a plain array, not a wrapped object.
func (c *HTTPClient) SubsidizedCommitteesByHeight(ctx context.Context, height uint64) ([]uint64, error) {
	if cached, ok := getCache[[]uint64](c, subsidizedCommitteesPath, height); ok {
		return cached, nil
	}
	var resp []uint64
	if err := c.doJSON(ctx, http.MethodPost, subsidizedCommitteesPath, QueryByHeightRequest{Height: height}, &resp); err != nil {
		return nil, err
	}
	setCache(c, subsidizedCommitteesPath, height, resp)
	return resp, nil
}

// RetiredCommitteesByHeight fetches retired committee IDs at the given height.
// Note: This endpoint returns a plain array, not a wrapped object.
func (c *HTTPClient) RetiredCommitteesByHeight(ctx context.Context, height uint64) ([]uint64, error) {
	if cached, ok := getCache[[]uint64](c, retiredCommitteesPath, height); ok {
		return cached, nil
	}
	var resp []uint64
	if err := c.doJSON(ctx, http.MethodPost, retiredCommitteesPath, QueryByHeightRequest{Height: height}, &resp); err != nil {
		return nil, err
	}
	setCache(c, retiredCommitteesPath, height, resp)
	return resp, nil
}

// SupplyByHeight fetches supply at the given height.
func (c *HTTPClient) SupplyByHeight(ctx context.Context, height uint64) (*fsm.Supply, error) {
	var resp fsm.Supply
	if err := c.doJSON(ctx, http.MethodPost, supplyPath, QueryByHeightRequest{Height: height}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Poll fetches the current poll.
func (c *HTTPClient) Poll(ctx context.Context) (fsm.Poll, error) {
	var resp fsm.Poll
	if err := c.doJSON(ctx, http.MethodGet, pollPath, nil, &resp); err != nil {
		return fsm.Poll{}, err
	}
	return resp, nil
}

// Proposals fetches governance proposals.
func (c *HTTPClient) Proposals(ctx context.Context) (fsm.GovProposals, error) {
	var resp fsm.GovProposals
	if err := c.doJSON(ctx, http.MethodGet, proposalPath, nil, &resp); err != nil {
		return fsm.GovProposals{}, err
	}
	return resp, nil
}

// pageResp is the response for a paged query.
type pageResp[T any] struct {
	PageNumber int `json:"pageNumber"`
	PerPage    int `json:"perPage"`
	Results    []T `json:"results"`
	Count      int `json:"count"`
	TotalPages int `json:"totalPages"`
	TotalCount int `json:"totalCount"`
}

// listPaged lists all pages of a given path.
func listPaged[T any](ctx context.Context, c *HTTPClient, path string, args any) ([]T, error) {
	var first pageResp[T]
	if err := c.doJSON(ctx, http.MethodPost, path, args, &first); err != nil {
		return nil, err
	}
	all := make([]T, 0, first.TotalCount)
	all = append(all, first.Results...)
	if first.TotalPages <= 1 {
		return all, nil
	}

	// Fetch remaining pages concurrently
	type res struct {
		items []T
		err   error
	}
	ch := make(chan res, first.TotalPages-1)
	for p := 2; p <= first.TotalPages; p++ {
		go func(page int) {
			var pr pageResp[T]
			payload := map[string]any{}
			if args != nil {
				if v, ok := args.(QueryByHeightRequest); ok {
					payload["height"] = v.Height
				}
			}
			payload["pageNumber"] = page
			if err := c.doJSON(ctx, http.MethodPost, path, payload, &pr); err != nil {
				ch <- res{nil, err}
				return
			}
			ch <- res{pr.Results, nil}
		}(p)
	}

	for i := 0; i < first.TotalPages-1; i++ {
		r := <-ch
		if r.err != nil {
			return nil, r.err
		}
		all = append(all, r.items...)
	}
	return all, nil
}
