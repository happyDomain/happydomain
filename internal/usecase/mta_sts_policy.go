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

package usecase

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
)

const (
	mtaStsBodySizeCap   = 64 * 1024
	mtaStsTotalDeadline = 10 * time.Second
	mtaStsConnTimeout   = 5 * time.Second
)

// FetchMTASTSPolicy implements happydns.ResolverUsecase.
func (us *resolverUsecase) FetchMTASTSPolicy(req happydns.MTASTSPolicyRequest) (*happydns.MTASTSPolicyResponse, error) {
	domain := strings.TrimSuffix(strings.TrimSpace(req.Domain), ".")
	if domain == "" {
		return nil, errors.New("domain is required")
	}

	host := "mta-sts." + dns.Fqdn(domain)
	host = strings.TrimSuffix(host, ".")
	url := "https://" + host + "/.well-known/mta-sts.txt"

	resp := &happydns.MTASTSPolicyResponse{URL: url}

	client := &http.Client{
		Timeout: mtaStsTotalDeadline,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: mtaStsConnTimeout,
			}).DialContext,
			TLSHandshakeTimeout:   mtaStsConnTimeout,
			ResponseHeaderTimeout: mtaStsConnTimeout,
		},
		// RFC 8461 sec. 3.3: receivers MUST NOT follow redirects when
		// fetching the policy file.
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	httpResp, err := client.Get(url)
	if err != nil {
		resp.Status, resp.ErrorMsg = classifyFetchError(err)
		return resp, nil
	}
	defer httpResp.Body.Close()

	resp.HTTPCode = httpResp.StatusCode

	body, readErr := io.ReadAll(io.LimitReader(httpResp.Body, mtaStsBodySizeCap+1))
	if readErr != nil {
		resp.Status = "fetch-error"
		resp.ErrorMsg = readErr.Error()
		return resp, nil
	}
	if len(body) > mtaStsBodySizeCap {
		resp.Status = "too-large"
		resp.ErrorMsg = fmt.Sprintf("body exceeds %d bytes", mtaStsBodySizeCap)
		resp.Body = string(body[:mtaStsBodySizeCap])
		return resp, nil
	}
	resp.Body = string(body)

	if httpResp.StatusCode >= 300 && httpResp.StatusCode < 400 {
		resp.Status = "http-error"
		resp.Redirected = true
		resp.ErrorMsg = fmt.Sprintf("server attempted a redirect (HTTP %d)", httpResp.StatusCode)
		return resp, nil
	}
	if httpResp.StatusCode == http.StatusNotFound {
		resp.Status = "not-found"
		resp.ErrorMsg = "no MTA-STS policy published at this URL"
		return resp, nil
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		resp.Status = "http-error"
		resp.ErrorMsg = fmt.Sprintf("HTTP %d", httpResp.StatusCode)
		return resp, nil
	}

	parseMTASTSBody(string(body), resp)
	resp.Status = "ok"
	return resp, nil
}

func classifyFetchError(err error) (status, msg string) {
	msg = err.Error()
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "fetch-error", "timeout: " + msg
	}
	var tlsErr *tls.CertificateVerificationError
	if errors.As(err, &tlsErr) {
		return "tls-error", msg
	}
	if strings.Contains(strings.ToLower(msg), "tls") || strings.Contains(strings.ToLower(msg), "certificate") {
		return "tls-error", msg
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "dns-error", msg
	}
	return "fetch-error", msg
}

// parseMTASTSBody parses the textual MTA-STS policy file (RFC 8461 sec. 3.2)
// and fills the policy fields of resp.
func parseMTASTSBody(body string, resp *happydns.MTASTSPolicyResponse) {
	// Lines may be terminated by CRLF or LF; trim both.
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, ':')
		if idx < 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:idx]))
		value := strings.TrimSpace(line[idx+1:])
		switch key {
		case "version":
			resp.Version = value
		case "mode":
			resp.Mode = strings.ToLower(value)
		case "mx":
			resp.MX = append(resp.MX, value)
		case "max_age":
			if n, err := strconv.Atoi(value); err == nil {
				resp.MaxAge = n
			}
		}
	}
}
