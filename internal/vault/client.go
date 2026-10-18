// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
// Authors: Pierre-Olivier Mercier, et al.
//
// This program is offered under a commercial and under the AGPL license.
// For commercial licensing, contact us at <contact@happydomain.org>.
//
// For AGPL licensing:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

// Package vault is a minimal client of the HashiCorp Vault HTTP API, and of
// OpenBao, which shares it: reading a version of a KV v2 secret, logging in
// with an AppRole, and renewing and looking up a token. It never repeats in
// an error what Vault answers, nor the path asked for, which may be a secret
// entered by mistake.
package vault

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
	"strings"
)

// maxResponse is the size of the largest answer read from Vault.
const maxResponse = 1 << 20

var (
	// ErrNotFound is returned for a secret, or version, that does not
	// exist, or is deleted.
	ErrNotFound = errors.New("no such secret in Vault")

	// ErrForbidden is returned when the token, or the AppRole, is refused.
	ErrForbidden = errors.New("Vault refused the credentials")

	// ErrNotRenewable is returned when Vault does not renew a token it
	// knows, as one that never expires.
	ErrNotRenewable = errors.New("Vault does not renew this token")
)

// Client talks to one Vault server.
type Client struct {
	base *url.URL
	http *http.Client

	// Namespace, when not empty, is sent with each request, for Vault
	// Enterprise and OpenBao namespaces.
	Namespace string
}

// New returns a client of the Vault server at addr, an http or https URL
// without path parameters nor credentials, reached through hc.
func New(addr string, hc *http.Client) (*Client, error) {
	u, err := url.Parse(addr)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("the Vault address is an http or https URL, such as https://vault.example:8200")
	}
	u.Path = strings.TrimSuffix(u.Path, "/")
	u.RawPath = ""
	return &Client{base: u, http: hc}, nil
}

// Address is the address of the server, as normalized.
func (c *Client) Address() string { return c.base.String() }

// Token is a token Vault handed out.
type Token struct {
	Token string

	// LeaseSeconds is how long the token lasts, 0 for ever.
	LeaseSeconds int

	Renewable bool
}

// escapePath escapes path segment by segment: its slashes separate KV folders
// and must not be escaped.
func escapePath(path string) string {
	segs := strings.Split(path, "/")
	for i, s := range segs {
		segs[i] = url.PathEscape(s)
	}
	return strings.Join(segs, "/")
}

// do sends a request to path, below /v1/, and decodes the answer into out. It
// returns the status of the answer, and ErrForbidden or ErrNotFound for
// those statuses.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, token string, body any, out any) (int, error) {
	u := *c.base
	// Path is derived from RawPath so that String keeps the escaping of each
	// segment, a "/" inside a segment included.
	u.RawPath = c.base.EscapedPath() + "/v1/" + path
	var err error
	if u.Path, err = url.PathUnescape(u.RawPath); err != nil {
		return 0, errors.New("invalid Vault path")
	}
	u.RawQuery = query.Encode()

	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), rd)
	if err != nil {
		return 0, errors.New("invalid Vault request")
	}
	if token != "" {
		req.Header.Set("X-Vault-Token", token)
	}
	if c.Namespace != "" {
		req.Header.Set("X-Vault-Namespace", c.Namespace)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		// The URL, which holds the path, is left out.
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err
		}
		return 0, fmt.Errorf("Vault cannot be reached: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse+1))
	if err != nil {
		return resp.StatusCode, errors.New("the answer of Vault cannot be read")
	}
	if len(raw) > maxResponse {
		return resp.StatusCode, errors.New("the answer of Vault is too large")
	}

	switch {
	case resp.StatusCode == http.StatusForbidden:
		return resp.StatusCode, ErrForbidden
	case resp.StatusCode == http.StatusNotFound:
		return resp.StatusCode, ErrNotFound
	case resp.StatusCode < 200 || resp.StatusCode > 299:
		return resp.StatusCode, fmt.Errorf("Vault answered with status %d", resp.StatusCode)
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return resp.StatusCode, errors.New("the answer of Vault does not decode")
		}
	}
	return resp.StatusCode, nil
}

// ReadKV2 returns the data of the secret at path, in the KV v2 engine mounted
// at mount: its version, or the latest when version is 0.
func (c *Client) ReadKV2(ctx context.Context, token, mount, path string, version int) (map[string]any, error) {
	var query url.Values
	if version > 0 {
		query = url.Values{"version": {strconv.Itoa(version)}}
	}
	var out struct {
		Data *struct {
			Data map[string]any `json:"data"`
		} `json:"data"`
	}
	if _, err := c.do(ctx, http.MethodGet, escapePath(mount)+"/data/"+escapePath(path), query, token, nil, &out); err != nil {
		return nil, err
	}
	if out.Data == nil || out.Data.Data == nil {
		return nil, ErrNotFound
	}
	return out.Data.Data, nil
}

type authAnswer struct {
	Auth *struct {
		ClientToken   string `json:"client_token"`
		LeaseDuration int    `json:"lease_duration"`
		Renewable     bool   `json:"renewable"`
	} `json:"auth"`
}

func (a authAnswer) token() (Token, error) {
	if a.Auth == nil || a.Auth.ClientToken == "" {
		return Token{}, errors.New("Vault answered no token")
	}
	return Token{Token: a.Auth.ClientToken, LeaseSeconds: a.Auth.LeaseDuration, Renewable: a.Auth.Renewable}, nil
}

// LoginAppRole logs in with the AppRole auth method mounted at mount.
func (c *Client) LoginAppRole(ctx context.Context, mount, roleID, secretID string) (Token, error) {
	var out authAnswer
	status, err := c.do(ctx, http.MethodPost, "auth/"+escapePath(mount)+"/login", nil, "", map[string]string{"role_id": roleID, "secret_id": secretID}, &out)
	if status == http.StatusBadRequest {
		// What Vault answers for a wrong role or secret ID.
		return Token{}, ErrForbidden
	}
	if err != nil {
		return Token{}, err
	}
	return out.token()
}

// RenewSelf renews token, for its usual lease.
func (c *Client) RenewSelf(ctx context.Context, token string) (Token, error) {
	var out authAnswer
	status, err := c.do(ctx, http.MethodPost, "auth/token/renew-self", nil, token, map[string]string{}, &out)
	if status == http.StatusBadRequest {
		// What Vault answers for a token whose lease is not renewable.
		return Token{}, ErrNotRenewable
	}
	if err != nil {
		return Token{}, err
	}
	return out.token()
}

// TokenInfo is what Vault tells of a token it knows.
type TokenInfo struct {
	// TTL is how many seconds the token has left, 0 for ever.
	TTL int

	Renewable bool

	// Period, when not 0, is the TTL each renewal gives back, with no
	// limit: the token is periodic.
	Period int

	// NumUses is how many requests the token has left, 0 for no limit.
	NumUses int

	// Type is service or batch.
	Type string
}

// LookupSelf returns what Vault knows of token, or ErrForbidden when Vault does
// not know it.
func (c *Client) LookupSelf(ctx context.Context, token string) (TokenInfo, error) {
	var out struct {
		Data *struct {
			TTL       int    `json:"ttl"`
			Renewable bool   `json:"renewable"`
			Period    int    `json:"period"`
			NumUses   int    `json:"num_uses"`
			Type      string `json:"type"`
		} `json:"data"`
	}
	if _, err := c.do(ctx, http.MethodGet, "auth/token/lookup-self", nil, token, nil, &out); err != nil {
		return TokenInfo{}, err
	}
	if out.Data == nil {
		return TokenInfo{}, errors.New("Vault answered nothing of the token")
	}
	d := out.Data
	return TokenInfo{TTL: d.TTL, Renewable: d.Renewable, Period: d.Period, NumUses: d.NumUses, Type: d.Type}, nil
}
