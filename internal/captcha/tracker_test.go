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

package captcha_test

import (
	"testing"
	"time"

	"git.happydns.org/happyDomain/internal/captcha"
)

func TestFailureTrackerThreshold(t *testing.T) {
	tracker := captcha.NewFailureTracker(3, time.Minute)
	defer tracker.Close()

	for i := 1; i < 3; i++ {
		tracker.RecordFailure("192.0.2.1", "user@example.com")
		if tracker.RequiresCaptcha("192.0.2.1", "user@example.com") {
			t.Fatalf("captcha required after %d failure(s), threshold is 3", i)
		}
	}

	tracker.RecordFailure("192.0.2.1", "user@example.com")
	if !tracker.RequiresCaptcha("192.0.2.1", "user@example.com") {
		t.Fatal("captcha not required after 3 failures")
	}

	tracker.RecordSuccess("192.0.2.1", "user@example.com")
	if tracker.RequiresCaptcha("192.0.2.1", "user@example.com") {
		t.Fatal("captcha still required after a successful login")
	}
}

// TestFailureTrackerSuccessOnlyCreditsOneAttempt covers the shared bucket case:
// the IP key is a network prefix, so a successful login must not hand back a
// whole threshold worth of attempts to everyone sharing it.
func TestFailureTrackerSuccessOnlyCreditsOneAttempt(t *testing.T) {
	tracker := captcha.NewFailureTracker(3, time.Minute)
	defer tracker.Close()

	// A neighbour on the same prefix sprays three accounts.
	tracker.RecordFailure("192.0.2.1", "alice@example.com")
	tracker.RecordFailure("192.0.2.1", "bob@example.com")
	tracker.RecordFailure("192.0.2.1", "carol@example.com")

	// A legitimate user on that prefix logs in.
	tracker.RecordSuccess("192.0.2.1", "dave@example.com")

	// That buys back exactly one attempt, not the whole window.
	if tracker.RequiresCaptcha("192.0.2.1", "erin@example.com") {
		t.Fatal("captcha required: the success should have credited one attempt")
	}

	tracker.RecordFailure("192.0.2.1", "erin@example.com")
	if !tracker.RequiresCaptcha("192.0.2.1", "frank@example.com") {
		t.Fatal("captcha not required: the success reset the shared counter")
	}
}

// TestFailureTrackerRepeatedSuccessesDrainTheCounter makes sure the credit is
// cumulative, so a user who mistyped their password does eventually get out of
// the captcha requirement without waiting for the window to elapse.
func TestFailureTrackerRepeatedSuccessesDrainTheCounter(t *testing.T) {
	tracker := captcha.NewFailureTracker(2, time.Minute)
	defer tracker.Close()

	for range 4 {
		tracker.RecordFailure("192.0.2.1", "user@example.com")
	}
	if !tracker.RequiresCaptcha("192.0.2.1", "user@example.com") {
		t.Fatal("captcha not required after 4 failures, threshold is 2")
	}

	tracker.RecordSuccess("192.0.2.1", "user@example.com")
	tracker.RecordSuccess("192.0.2.1", "user@example.com")
	if !tracker.RequiresCaptcha("192.0.2.1", "user@example.com") {
		t.Fatal("captcha not required with 2 failures left, threshold is 2")
	}

	tracker.RecordSuccess("192.0.2.1", "user@example.com")
	if tracker.RequiresCaptcha("192.0.2.1", "user@example.com") {
		t.Fatal("captcha still required after the counter was drained below the threshold")
	}
}

// TestFailureTrackerSuccessOnUnknownKey guards against a success for a source
// that has no recorded failure resurrecting or corrupting an entry.
func TestFailureTrackerSuccessOnUnknownKey(t *testing.T) {
	tracker := captcha.NewFailureTracker(1, time.Minute)
	defer tracker.Close()

	tracker.RecordSuccess("192.0.2.1", "user@example.com")

	tracker.RecordFailure("192.0.2.1", "user@example.com")
	if !tracker.RequiresCaptcha("192.0.2.1", "user@example.com") {
		t.Fatal("captcha not required after reaching the threshold")
	}
}

// TestFailureTrackerSprayingIsCaught covers the password spraying case: one
// source, many target accounts. The IP half of the tracker must trip even
// though no single email reaches the threshold.
func TestFailureTrackerSprayingIsCaught(t *testing.T) {
	tracker := captcha.NewFailureTracker(3, time.Minute)
	defer tracker.Close()

	tracker.RecordFailure("192.0.2.1", "alice@example.com")
	tracker.RecordFailure("192.0.2.1", "bob@example.com")
	tracker.RecordFailure("192.0.2.1", "carol@example.com")

	if !tracker.RequiresCaptcha("192.0.2.1", "dave@example.com") {
		t.Fatal("captcha not required for a fourth account tried from the same source")
	}
	if tracker.RequiresCaptcha("198.51.100.1", "dave@example.com") {
		t.Fatal("captcha required for an unrelated source")
	}
}

// TestFailureTrackerEmailIsNormalized makes sure the email half of the lockout
// cannot be reset by changing the case or padding the address.
func TestFailureTrackerEmailIsNormalized(t *testing.T) {
	tracker := captcha.NewFailureTracker(2, time.Minute)
	defer tracker.Close()

	tracker.RecordFailure("192.0.2.1", "user@example.com")
	tracker.RecordFailure("198.51.100.1", " User@Example.COM ")

	if !tracker.RequiresCaptcha("203.0.113.1", "USER@EXAMPLE.COM") {
		t.Fatal("captcha not required: the email variants were counted separately")
	}
}

func TestFailureTrackerWindowExpires(t *testing.T) {
	tracker := captcha.NewFailureTracker(1, 10*time.Millisecond)
	defer tracker.Close()

	tracker.RecordFailure("192.0.2.1", "user@example.com")
	if !tracker.RequiresCaptcha("192.0.2.1", "user@example.com") {
		t.Fatal("captcha not required right after reaching the threshold")
	}

	time.Sleep(20 * time.Millisecond)

	if tracker.RequiresCaptcha("192.0.2.1", "user@example.com") {
		t.Fatal("captcha still required after the window elapsed")
	}
}
