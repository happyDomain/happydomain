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

package secret

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"git.happydns.org/happyDomain/model"
)

func TestUserErrorIgnoresOtherErrors(t *testing.T) {
	for name, err := range map[string]error{
		"nil":       nil,
		"unrelated": errors.New("storage down"),
		"redacted":  ErrRedactedSecret,
	} {
		if got := UserError(err, "provider"); got != nil {
			t.Errorf("UserError(%s) = %v, want nil", name, got)
		}
	}
}

func TestUserErrorTellsWhatToDo(t *testing.T) {
	// The reasons name a safe: not the user's to see.
	const reason = "safe 9UQuy5DstzVHBom1IHwYdg"

	for name, c := range map[string]struct {
		err      error
		sentinel error
		status   int
		hint     string
	}{
		"unopenable":       {unopenable(errors.New(reason)), ErrUnopenable, http.StatusBadRequest, "enter it again"},
		"safe unavailable": {safeUnavailable(errors.New(reason)), ErrSafeUnavailable, http.StatusServiceUnavailable, "administrator"},
	} {
		t.Run(name, func(t *testing.T) {
			// Wrapped on the way, as callers do.
			got := UserError(fmt.Errorf("apikey: %w", c.err), "provider")
			if got == nil {
				t.Fatal("UserError = nil")
			}

			var he happydns.HTTPError
			if !errors.As(got, &he) {
				t.Fatalf("UserError = %T, want a happydns.HTTPError", got)
			}
			if he.HTTPStatus() != c.status {
				t.Errorf("HTTPStatus() = %d, want %d", he.HTTPStatus(), c.status)
			}

			msg := he.ToErrorResponse().Message
			if msg != got.Error() {
				t.Errorf("response %q differs from Error() %q", msg, got.Error())
			}
			if !strings.Contains(msg, "provider") || !strings.Contains(msg, c.hint) {
				t.Errorf("message %q does not name the object and %q", msg, c.hint)
			}
			if strings.Contains(msg, "9UQuy5DstzVHBom1IHwYdg") {
				t.Errorf("message %q leaks the reason", msg)
			}

			if !errors.Is(got, c.sentinel) {
				t.Error("the reason is no longer reachable with errors.Is")
			}

			// Translated already and wrapped again: kept as it was.
			again := UserError(fmt.Errorf("unable to retrieve the zone: %w", got), "domain")
			if again == nil || again.Error() != got.Error() {
				t.Errorf("UserError again = %v, want %v", again, got)
			}
		})
	}
}
