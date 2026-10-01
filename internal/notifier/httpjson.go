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

package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"git.happydns.org/happyDomain/model"
)

// Shared by both webhook and UnifiedPush.
type httpJSONPayload struct {
	Event        string                `json:"event"`
	Checker      string                `json:"checker"`
	Domain       string                `json:"domain"`
	Target       happydns.CheckTarget  `json:"target"`
	OldStatus    happydns.Status       `json:"oldStatus"`
	NewStatus    happydns.Status       `json:"newStatus"`
	States       []happydns.CheckState `json:"states,omitempty"`
	Timestamp    time.Time             `json:"timestamp"`
	DashboardURL string                `json:"dashboardUrl,omitempty"`
}

func buildHTTPPayload(p *NotificationPayload, dashboardURL string) httpJSONPayload {
	return httpJSONPayload{
		Event:        "status_change",
		Checker:      p.CheckerID,
		Domain:       p.DomainName,
		Target:       p.Target,
		OldStatus:    p.OldStatus,
		NewStatus:    p.NewStatus,
		States:       p.States,
		Timestamp:    time.Now(),
		DashboardURL: dashboardURL,
	}
}

// decorate runs after marshal so it can sign the exact bytes (e.g. HMAC).
// maxResponseBodyBytes bounds the drain below: the body is unused, we only read
// it so the connection can be reused.
const maxResponseBodyBytes = 64 * 1024

func postJSON(ctx context.Context, client *http.Client, endpoint string, body any, decorate func(*http.Request, []byte)) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("creating request: %w", hideURL(err))
	}
	req.Header.Set("Content-Type", "application/json")
	if decorate != nil {
		decorate(req, raw)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", hideURL(err))
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBodyBytes))

	if resp.StatusCode >= 300 {
		return fmt.Errorf("endpoint returned status %d", resp.StatusCode)
	}
	return nil
}

// hideURL returns err, a *url.Error as the HTTP client returns, with its URL
// cut down to scheme and host. The rest may be a capability (a UnifiedPush
// endpoint, the token of a chat webhook), and a failed send ends up in the
// logs and in the notification records stored in the database. What err
// wraps is kept, so that errors.Is still sees a timeout.
func hideURL(err error) error {
	ue, ok := err.(*url.Error)
	if !ok {
		return err
	}

	short := happydns.RedactedSecret
	if u, perr := url.Parse(ue.URL); perr == nil && u.Host != "" {
		short = u.Scheme + "://" + u.Host + "/" + happydns.RedactedSecret
	}

	return &url.Error{Op: ue.Op, URL: short, Err: ue.Err}
}

func testPayload(rcpt Recipient) *NotificationPayload {
	return &NotificationPayload{
		Recipient:  rcpt,
		CheckerID:  "test",
		DomainName: "example.com",
		OldStatus:  happydns.StatusOK,
		NewStatus:  happydns.StatusWarn,
		Annotation: "This is a test notification from happyDomain.",
	}
}
