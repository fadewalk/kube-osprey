package alertmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const requestTimeout = 15 * time.Second

type Client struct {
	base string
	http *http.Client
}

func NewClient(alertManagerURL string, rt http.RoundTripper) (*Client, error) {
	alertManagerURL = strings.TrimRight(alertManagerURL, "/")
	if alertManagerURL == "" {
		return nil, fmt.Errorf("alertmanager URL cannot be empty")
	}
	if _, err := url.Parse(alertManagerURL); err != nil {
		return nil, fmt.Errorf("invalid alertmanager URL: %w", err)
	}
	return &Client{
		base: alertManagerURL,
		http: &http.Client{Timeout: requestTimeout, Transport: rt},
	}, nil
}

type Alert struct {
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL,omitempty"`
	Fingerprint  string            `json:"fingerprint,omitempty"`
	Status       AlertStatus       `json:"status"`
}

type AlertStatus struct {
	State       string   `json:"state"`
	SilencedBy  []string `json:"silencedBy,omitempty"`
	InhibitedBy []string `json:"inhibitedBy,omitempty"`
}

type Matcher struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	IsRegex  bool   `json:"isRegex"`
	Operator string `json:"operator,omitempty"`
}

type Silence struct {
	ID        string        `json:"id"`
	Matchers  []Matcher     `json:"matchers"`
	StartsAt  time.Time     `json:"startsAt"`
	EndsAt    time.Time     `json:"endsAt"`
	CreatedAt time.Time     `json:"createdAt,omitempty"`
	CreatedBy string        `json:"createdBy,omitempty"`
	Comment   string        `json:"comment,omitempty"`
	Status    SilenceStatus `json:"status"`
}

type SilenceStatus struct {
	State string `json:"state"`
}

type SilencePayload struct {
	Matchers  []Matcher `json:"matchers" binding:"required,min=1"`
	StartsAt  time.Time `json:"startsAt"`
	EndsAt    time.Time `json:"endsAt"`
	CreatedBy string    `json:"createdBy,omitempty"`
	Comment   string    `json:"comment"`
}

type Status struct {
	Version     string `json:"version"`
	ClusterName string `json:"clusterName,omitempty"`
	Peers       int    `json:"peers"`
}

type alertmanagerStatusResponse struct {
	VersionInfo struct {
		Version string `json:"version"`
	} `json:"versionInfo"`
	Cluster struct {
		Name   string `json:"name"`
		Status string `json:"status"`
		Peers  []struct {
			Name string `json:"name"`
		} `json:"peers"`
	} `json:"cluster"`
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to encode request body: %w", err)
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("alertmanager API %s %s: %s", method, path, strings.TrimSpace(string(respBody)))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) ListAlerts(ctx context.Context) ([]Alert, error) {
	var alerts []Alert
	if err := c.do(ctx, http.MethodGet, "/api/v2/alerts", nil, &alerts); err != nil {
		return nil, err
	}
	return alerts, nil
}

func (c *Client) ListSilences(ctx context.Context) ([]Silence, error) {
	var silences []Silence
	if err := c.do(ctx, http.MethodGet, "/api/v2/silences", nil, &silences); err != nil {
		return nil, err
	}
	return silences, nil
}

func (c *Client) CreateSilence(ctx context.Context, payload SilencePayload) (string, error) {
	var result struct {
		SilenceID string `json:"silenceID"`
	}
	if err := c.do(ctx, http.MethodPost, "/api/v2/silences", payload, &result); err != nil {
		return "", err
	}
	return result.SilenceID, nil
}

func (c *Client) DeleteSilence(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/api/v2/silence/"+url.PathEscape(id), nil, nil)
}

func (c *Client) GetStatus(ctx context.Context) (*Status, error) {
	var raw alertmanagerStatusResponse
	if err := c.do(ctx, http.MethodGet, "/api/v2/status", nil, &raw); err != nil {
		return nil, err
	}
	return &Status{
		Version:     raw.VersionInfo.Version,
		ClusterName: raw.Cluster.Name,
		Peers:       len(raw.Cluster.Peers),
	}, nil
}
