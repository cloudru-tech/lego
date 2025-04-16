package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/providers/dns/internal/errutils"
)

type AuthRequest struct {
	KeyID  string `json:"keyId"`
	Secret string `json:"secret"`
}

type token string

const tokenKey token = "token"

// obtainToken Logs into cloud.ru and acquires a bearer token for use in future API calls.
// https://cloud.ru/ru/docs/clouddns/ug/topics/api-ref_authentication.html
func (c *Client) obtainToken(ctx context.Context) (*Token, error) {
	data := AuthRequest{
		KeyID:  c.keyID,
		Secret: c.secret,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.AuthEndpoint.String(),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, errutils.NewHTTPDoError(req, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(req, resp)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errutils.NewReadResponseError(req, resp.StatusCode, err)
	}

	tok := Token{}
	err = json.Unmarshal(raw, &tok)
	if err != nil {
		return nil, errutils.NewUnmarshalError(req, resp.StatusCode, raw, err)
	}

	if !strings.EqualFold(tok.TokenType, "Bearer") {
		return nil, fmt.Errorf("received unexpected token type: %s", tok.TokenType)
	}

	tok.Deadline = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)

	return &tok, nil
}

func (c *Client) CreateAuthenticatedContext(ctx context.Context) (context.Context, error) {
	c.muToken.Lock()
	defer c.muToken.Unlock()

	if c.token != nil && time.Now().Before(c.token.Deadline) {
		// Already authenticated, stop now
		return context.WithValue(ctx, tokenKey, c.token), nil
	}

	tok, err := c.obtainToken(ctx)
	if err != nil {
		return nil, err
	}

	return context.WithValue(ctx, tokenKey, tok), nil
}

func parseError(req *http.Request, resp *http.Response) error {
	if resp.StatusCode < 400 || resp.StatusCode > 499 {
		return errutils.NewUnexpectedResponseStatusCodeError(req, resp)
	}

	raw, _ := io.ReadAll(resp.Body)

	errResp := &authResponseError{}
	err := json.Unmarshal(raw, errResp)
	if err != nil {
		return errutils.NewUnexpectedStatusCodeError(req, resp.StatusCode, raw)
	}

	return fmt.Errorf("%d: %w", resp.StatusCode, errResp)
}

func getToken(ctx context.Context) *Token {
	tok, ok := ctx.Value(tokenKey).(*Token)
	if !ok {
		return nil
	}

	return tok
}
