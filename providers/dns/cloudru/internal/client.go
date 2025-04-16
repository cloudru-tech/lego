package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/go-acme/lego/v4/providers/dns/internal/errutils"
)

// Default API endpoints.
const (
	APIBaseURL  = "https://console.cloud.ru/u-api/svp/evodns/v1"
	AuthBaseURL = "https://iam.api.cloud.ru/api/v1/auth/token"
)

// Client the Cloud.ru API client.
type Client struct {
	keyID  string
	secret string

	APIEndpoint  *url.URL
	AuthEndpoint *url.URL
	HTTPClient   *http.Client

	token   *Token
	muToken sync.Mutex
}

// NewClient Creates a new Client.
func NewClient(login, secret string) *Client {
	apiEndpoint, _ := url.Parse(APIBaseURL)
	authEndpoint, _ := url.Parse(AuthBaseURL)

	return &Client{
		keyID:        login,
		secret:       secret,
		APIEndpoint:  apiEndpoint,
		AuthEndpoint: authEndpoint,
		HTTPClient:   &http.Client{Timeout: 5 * time.Second},
	}
}

// GetZones returns all zones available for account.
func (c *Client) GetZones(ctx context.Context, projectID string) ([]Zone, error) {
	var allZones []Zone
	pageSize := 10
	offset := 0
	endpoint := c.APIEndpoint.JoinPath("zones")
	query := endpoint.Query()
	query.Set("projectId", projectID)

	for {
		query.Set("page", strconv.Itoa(pageSize))
		query.Set("offset", strconv.Itoa(offset))
		endpoint.RawQuery = query.Encode()

		req, err := newJSONRequest(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}

		var zones ZonesAPIResponse
		err = c.do(req, &zones)
		if err != nil {
			return nil, err
		}

		allZones = append(allZones, zones.Zones...)

		if zones.Total < offset+pageSize {
			break
		}
		offset += pageSize
	}

	return allZones, nil
}

func (c *Client) GetRecords(ctx context.Context, zoneID string) ([]Record, error) {
	endpoint := c.APIEndpoint.JoinPath("public", "records")
	var allRecords []Record
	pageSize := 10
	offset := 0
	query := endpoint.Query()
	query.Set("zoneId", zoneID)

	for {
		query.Set("page", strconv.Itoa(pageSize))
		query.Set("offset", strconv.Itoa(offset))
		endpoint.RawQuery = query.Encode()

		req, err := newJSONRequest(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}

		var records RecordsAPIResponse
		err = c.do(req, &records)
		if err != nil {
			return nil, err
		}

		allRecords = append(allRecords, records.Records...)

		if records.Total < offset+pageSize {
			break
		}
		offset += pageSize
	}

	return allRecords, nil
}

func (c *Client) CreateRecord(ctx context.Context, request CreateRecordRequest) error {
	endpoint := c.APIEndpoint.JoinPath("public", "records")

	req, err := newJSONRequest(ctx, http.MethodPost, endpoint, request)
	if err != nil {
		return err
	}

	err = c.do(req, nil)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) DeleteRecord(ctx context.Context, recordID string) error {
	endpoint := c.APIEndpoint.JoinPath("public", "records", recordID)

	req, err := newJSONRequest(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}

	return c.do(req, nil)
}

func (c *Client) FindRecordByName(ctx context.Context, zoneID, recordName string) (*Record, error) {
	records, err := c.GetRecords(ctx, zoneID)
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		if record.Name == recordName {
			return &record, nil
		}
	}
	return nil, errors.New("record by name not found")
}

func (c *Client) do(req *http.Request, result any) error {
	tok := getToken(req.Context())
	if tok != nil {
		req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	} else {
		return errors.New("not logged in")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return errutils.NewHTTPDoError(req, err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return errutils.NewUnexpectedResponseStatusCodeError(req, resp)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return errutils.NewReadResponseError(req, resp.StatusCode, err)
	}

	if result == nil {
		return nil
	}

	err = json.Unmarshal(raw, result)
	if err != nil {
		return errutils.NewUnmarshalError(req, resp.StatusCode, raw, err)
	}

	return nil
}

func newJSONRequest(ctx context.Context, method string, endpoint *url.URL, payload any) (*http.Request, error) {
	buf := new(bytes.Buffer)

	if payload != nil {
		err := json.NewEncoder(buf).Encode(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to create request JSON body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), buf)
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}
