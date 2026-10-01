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
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

type failingTransport struct{ err error }

func (t failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, t.err
}

// A failed send must not carry the URL it was sent to: a UnifiedPush endpoint
// is a capability, sealed in the channel, and the error ends up in the logs
// and in the notification records stored in the database.
func TestPostJSONErrorHidesTheURL(t *testing.T) {
	client := &http.Client{Transport: failingTransport{err: context.DeadlineExceeded}}

	for _, endpoint := range []string{
		"https://push.example.net/up/s3cr3t-token",
		"https://push.example.net/up?token=s3cr3t-token",
		"https://user:s3cr3t-token@push.example.net/up",
	} {
		err := postJSON(context.Background(), client, endpoint, map[string]string{}, nil)
		if err == nil {
			t.Fatalf("postJSON(%q) = nil, want an error", endpoint)
		}
		if strings.Contains(err.Error(), "s3cr3t-token") {
			t.Errorf("postJSON(%q) error reveals the secret: %v", endpoint, err)
		}
		if !strings.Contains(err.Error(), "push.example.net") {
			t.Errorf("postJSON(%q) error does not name the host anymore: %v", endpoint, err)
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("postJSON(%q) error lost its cause: %v", endpoint, err)
		}
	}
}

// Same when the URL does not even parse.
func TestPostJSONInvalidURLErrorHidesTheURL(t *testing.T) {
	client := &http.Client{Transport: failingTransport{err: errors.New("not reached")}}

	err := postJSON(context.Background(), client, "https://push.example.net/up/s3cr3t-token\x7f", map[string]string{}, nil)
	if err == nil {
		t.Fatal("postJSON with an invalid URL = nil, want an error")
	}
	if strings.Contains(err.Error(), "s3cr3t-token") {
		t.Errorf("postJSON error reveals the secret: %v", err)
	}
}
