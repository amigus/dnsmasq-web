package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
)

// Client wraps resty and handles token injection, retry-on-401, and error handling.
type Client struct {
	config        *Config
	tokenProvider *TokenProvider
	restyClient   *resty.Client
	currentToken  string
}

// NewClient creates a new HTTP client with the given configuration.
func NewClient(cfg *Config) (*Client, error) {
	cfg.ResolveSSHServer()

	tokenProvider := NewTokenProvider(cfg)

	// For remote servers or when noAuth is false, try to get an initial token
	// If token retrieval fails, we'll try again on the first actual request
	var initialToken string
	if !cfg.NoAuth {
		token, err := tokenProvider.GetToken()
		if err != nil {
			// Don't fail if token retrieval fails - we'll try again on first request
			// This allows the CLI to show help even when server is down
		} else {
			initialToken = token
		}
	}

	restyClient := resty.New().
		SetBaseURL(fmt.Sprintf("http://%s", cfg.Server)).
		SetTimeout(30 * time.Second).
		SetRetryCount(2).
		SetRetryWaitTime(time.Second)

	c := &Client{
		config:        cfg,
		tokenProvider: tokenProvider,
		restyClient:   restyClient,
		currentToken:  initialToken,
	}

	// Add auth header if not skipping auth and we have a token
	if !cfg.NoAuth {
		if initialToken != "" {
			c.restyClient.SetHeader("X-Token", initialToken)
		}
		c.restyClient.AddRetryCondition(c.retryCondition)
	}

	return c, nil
}

// retryCondition handles 401 Unauthorized by refreshing the token and retrying.
// It also fetches the initial token if we don't have one yet.
func (c *Client) retryCondition(r *resty.Response, _ error) bool {
	if !c.config.NoAuth {
		// If we need a token and don't have one, try to get it
		if c.currentToken == "" {
			newToken, err := c.tokenProvider.GetToken()
			if err == nil && newToken != "" {
				c.currentToken = newToken
				c.restyClient.SetHeader("X-Token", newToken)
				return true // Retry the request
			}
		} else if r.StatusCode() == http.StatusUnauthorized {
			// Token was rejected, try to refresh it
			newToken, err := c.tokenProvider.GetToken()
			if err == nil && newToken != "" && newToken != c.currentToken {
				c.currentToken = newToken
				c.restyClient.SetHeader("X-Token", newToken)
				return true // Retry the request
			}
		}
	}
	return false
}

// do performs an HTTP request and handles error responses.
func (c *Client) do(method, endpoint string, body any, result any) error {
	req := c.restyClient.R()

	if body != nil {
		req.SetHeader("Content-Type", "application/json")
		req.SetBody(body)
	}

	resp, err := req.Execute(method, endpoint)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	// Handle 4xx and 5xx responses
	if resp.StatusCode() >= 400 {
		// Try to parse error response
		var errResp map[string]any
		if err := json.Unmarshal(resp.Body(), &errResp); err == nil {
			if errMsg, ok := errResp["error"]; ok {
				return fmt.Errorf("server error (HTTP %d): %v", resp.StatusCode(), errMsg)
			}
		}
		return fmt.Errorf("server error (HTTP %d): %s", resp.StatusCode(), resp.String())
	}

	// Unmarshal response into result if provided
	if result != nil {
		if err := json.Unmarshal(resp.Body(), result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// GetLeases fetches all active DHCP leases.
func (c *Client) GetLeases() ([]Lease, error) {
	var leases []Lease
	if err := c.do("GET", "/leases", nil, &leases); err != nil {
		return nil, err
	}
	return leases, nil
}

// GetClients fetches all DHCP clients with optional since filter.
func (c *Client) GetClients(since string) ([]ClientInfo, error) {
	endpoint := "/clients"
	if since != "" {
		endpoint = fmt.Sprintf("/clients?since=%s", since)
	}
	var clients []ClientInfo
	if err := c.do("GET", endpoint, nil, &clients); err != nil {
		return nil, err
	}
	return clients, nil
}

// GetAddresses fetches address history for a given MAC.
func (c *Client) GetAddresses(mac string) ([]IPHistory, error) {
	var history []IPHistory
	endpoint := fmt.Sprintf("/addresses/%s", mac)
	if err := c.do("GET", endpoint, nil, &history); err != nil {
		return nil, err
	}
	return history, nil
}

// GetDevices fetches device (MAC) history for a given IPv4.
func (c *Client) GetDevices(ipv4 string) ([]MacHistory, error) {
	var history []MacHistory
	endpoint := fmt.Sprintf("/devices/%s", ipv4)
	if err := c.do("GET", endpoint, nil, &history); err != nil {
		return nil, err
	}
	return history, nil
}

// GetRequests fetches request history for a given CIDR or range.
func (c *Client) GetRequests(cidr, ipRange string) (map[string][]Request, error) {
	var result map[string][]Request
	var endpoint string

	if cidr != "" {
		endpoint = fmt.Sprintf("/requests?cidr=%s", cidr)
	} else if ipRange != "" {
		endpoint = fmt.Sprintf("/requests?range=%s", ipRange)
	} else {
		return nil, fmt.Errorf("either cidr or range must be specified")
	}

	if err := c.do("GET", endpoint, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetReservations fetches all reservations.
func (c *Client) GetReservations() ([]Reservation, error) {
	var reservations []Reservation
	if err := c.do("GET", "/reservations", nil, &reservations); err != nil {
		return nil, err
	}
	return reservations, nil
}

// GetReservation fetches a single reservation by MAC.
func (c *Client) GetReservation(mac string) (*Reservation, error) {
	var res Reservation
	endpoint := fmt.Sprintf("/reservations/%s", mac)
	if err := c.do("GET", endpoint, nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// CreateReservation creates a new reservation.
func (c *Client) CreateReservation(mac, ipv4, hostname string, tags []string, leaseTime string) (*Reservation, error) {
	req := map[string]any{
		"mac":  mac,
		"ipv4": ipv4,
	}
	if hostname != "" {
		req["hostname"] = hostname
	}
	if len(tags) > 0 {
		req["tags"] = tags
	}
	if leaseTime != "" {
		req["lease_time"] = leaseTime
	}

	if err := c.do("POST", "/reservations", req, nil); err != nil {
		return nil, err
	}

	// Fetch and return the created reservation
	return c.GetReservation(mac)
}

// PinReservation creates a reservation from the current DHCP lease.
func (c *Client) PinReservation(mac, hostname string, tags []string, leaseTime string, force bool) (*Reservation, error) {
	req := map[string]any{
		"mac":        mac,
		"from_lease": true,
	}
	if hostname != "" {
		req["hostname"] = hostname
	}
	if len(tags) > 0 {
		req["tags"] = tags
	}
	if leaseTime != "" {
		req["lease_time"] = leaseTime
	}

	endpoint := "/reservations"
	if force {
		endpoint += "?force=1"
	}
	if err := c.do("POST", endpoint, req, nil); err != nil {
		return nil, err
	}
	return c.GetReservation(mac)
}

// UpdateReservation updates an existing reservation by MAC.
func (c *Client) UpdateReservation(mac string, update map[string]any) (*Reservation, error) {
	endpoint := fmt.Sprintf("/reservations/%s", mac)
	if err := c.do("PUT", endpoint, update, nil); err != nil {
		return nil, err
	}

	// Fetch and return the updated reservation
	return c.GetReservation(mac)
}

// DeleteReservation deletes a reservation by MAC.
func (c *Client) DeleteReservation(mac string) error {
	endpoint := fmt.Sprintf("/reservations/%s", mac)
	return c.do("DELETE", endpoint, nil, nil)
}
