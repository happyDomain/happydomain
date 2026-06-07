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

package database

import (
	"fmt"
	"testing"
	"time"

	happydns "git.happydns.org/happyDomain/model"
)

// maxID is the longest possible Identifier (16 bytes of 0xFF) whose
// base64url encoding is 22 chars.
var maxID = happydns.Identifier{
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
}

// maxIDPtr is a pointer to maxID for functions that take *Identifier.
var maxIDPtr = &maxID

// worstTime exercises the reverseChronoSegment path; the encoding is fixed-width
// so any time produces the same length, but the epoch is a natural sentinel.
var worstTime = time.Unix(0, 0)

func assertKeySize(t *testing.T, name, key string) {
	t.Helper()
	if n := len(key); n > maxKeySize {
		t.Errorf("%s: key length %d exceeds limit %d\n  key = %q", name, n, maxKeySize, key)
	}
}

// --- auth ---

func TestAuthPrimaryKeySize(t *testing.T) {
	assertKeySize(t, "authPrimaryKey", authPrimaryKey(maxID))
}

func TestAuthEmailIndexKeySize(t *testing.T) {
	// FQDN is hashed; test with an arbitrarily long email.
	assertKeySize(t, "authEmailIndexKey", authEmailIndexKey("very-long-email-address-that-is-unrealistically-long@example.com"))
}

// --- user ---

func TestUserPrimaryKeySize(t *testing.T) {
	assertKeySize(t, "userPrimaryKey", userPrimaryKey(maxID))
}

func TestUserEmailIndexKeySize(t *testing.T) {
	assertKeySize(t, "userEmailIndexKey", userEmailIndexKey("very-long-email-address-that-is-unrealistically-long@example.com"))
}

// --- session ---

func TestSessionPrimaryKeySize(t *testing.T) {
	assertKeySize(t, "sessionKey", sessionKey("arbitrary-long-raw-session-id"))
}

func TestSessionUserIndexKeySize(t *testing.T) {
	fullHash := sessionHash("arbitrary-long-raw-session-id")
	key := sessionUserIndexKey(maxID, sessionShortHash(fullHash))
	assertKeySize(t, "sessionUserIndexKey", key)
}

// --- domain ---

func TestDomainPrimaryKeySize(t *testing.T) {
	assertKeySize(t, "domainPrimaryKey", fmt.Sprintf("%s%s", domainPrimaryPrefix, maxID.String()))
}

func TestDomainOwnerIndexKeySize(t *testing.T) {
	assertKeySize(t, "domainOwnerIndexKey", domainOwnerIndexKey(maxID, maxID))
}

func TestDomainFQDNIndexKeySize(t *testing.T) {
	// FQDN is hashed so the key length is independent of domain name length.
	assertKeySize(t, "domainFQDNIndexKey", domainFQDNIndexKey("very.long.domain.name.that.is.unrealistically.long.example.com", maxID))
}

// --- domain log ---

func TestDomainLogKeySize(t *testing.T) {
	key := fmt.Sprintf("%s%s|%s", domainLogPrimaryPrefix, maxID.String(), maxID.String())
	assertKeySize(t, "domainLogKey", key)
}

// --- zone ---

func TestZonePrimaryKeySize(t *testing.T) {
	assertKeySize(t, "zonePrimaryKey", fmt.Sprintf("%s%s", zonePrimaryPrefix, maxID.String()))
}

// --- provider ---

func TestProviderPrimaryKeySize(t *testing.T) {
	assertKeySize(t, "providerPrimaryKey", fmt.Sprintf("%s%s", providerPrimaryPrefix, maxID.String()))
}

func TestProviderOwnerKeySize(t *testing.T) {
	assertKeySize(t, "providerOwnerKey", providerOwnerKey(maxID, maxID))
}

// --- check plan ---

func TestCheckPlanPrimaryKeySize(t *testing.T) {
	assertKeySize(t, "checkPlanPrimaryKey", fmt.Sprintf("%s%s", checkPlanPrimaryPrefix, maxID.String()))
}

func TestPlanUserIndexKeySize(t *testing.T) {
	assertKeySize(t, "planUserIndexKey", planUserIndexKey(maxID.String(), maxID.String()))
}

// --- check evaluation ---

func TestEvaluationPrimaryKeySize(t *testing.T) {
	assertKeySize(t, "evaluationPrimaryKey", fmt.Sprintf("%s%s", evaluationPrimaryPrefix, maxID.String()))
}

func TestEvaluationPlanIndexKeySize(t *testing.T) {
	assertKeySize(t, "evaluationPlanIndexKey", evaluationPlanIndexKey(maxID.String(), worstTime, maxID.String()))
}

// --- execution ---

func TestExecutionPrimaryKeySize(t *testing.T) {
	assertKeySize(t, "executionPrimaryKey", fmt.Sprintf("%s%s", ExecutionPrimaryPrefix, maxID.String()))
}

func TestExecutionPlanIndexKeySize(t *testing.T) {
	key := fmt.Sprintf("%s%s|%s", ExecutionByPlanIndexPrefix, maxID.String(), maxID.String())
	assertKeySize(t, "executionPlanIndexKey", key)
}

func TestExecutionUserIndexKeySize(t *testing.T) {
	assertKeySize(t, "executionUserIndexKey", executionUserIndexKey(maxID.String(), worstTime, maxID.String()))
}

func TestExecutionDomainIndexKeySize(t *testing.T) {
	assertKeySize(t, "executionDomainIndexKey", executionDomainIndexKey(maxID.String(), worstTime, maxID.String()))
}

// --- notification channel ---

func TestNotifchPrimaryKeySize(t *testing.T) {
	assertKeySize(t, "notifchPrimaryKey", notifchPrimaryKey(maxID))
}

func TestNotifchUserKeySize(t *testing.T) {
	assertKeySize(t, "notifchUserKey", notifchUserKey(maxID, maxID))
}

// --- notification preference ---

func TestNotifprefPrimaryKeySize(t *testing.T) {
	assertKeySize(t, "notifprefPrimaryKey", notifprefPrimaryKey(maxID))
}

func TestNotifprefUserKeySize(t *testing.T) {
	assertKeySize(t, "notifprefUserKey", notifprefUserKey(maxID, maxID))
}

// --- notification record ---

func TestNotifrecPrimaryKeySize(t *testing.T) {
	assertKeySize(t, "notifrecPrimaryKey", notifrecPrimaryKey(maxID))
}

func TestNotifrecUserKeySize(t *testing.T) {
	assertKeySize(t, "notifrecUserKey", notifrecUserKey(maxID, maxID))
}

// --- notification state ---

func TestNotifStateKeySize(t *testing.T) {
	// checkerID and target fields are hashed so key length is bounded.
	target := happydns.CheckTarget{
		UserId:    maxID.String(),
		DomainId:  maxID.String(),
		ServiceId: maxID.String(),
	}
	assertKeySize(t, "notifStateKey", notifStateKey("a-very-long-checker-name.v99", target, maxID))
}

// --- observation snapshot ---

func TestObservationSnapshotKeySize(t *testing.T) {
	assertKeySize(t, "observationSnapshotKey", fmt.Sprintf("%s%s", observationSnapshotPrefix, maxID.String()))
}

// --- scheduler ---

func TestSchedulerLastRunKeySize(t *testing.T) {
	assertKeySize(t, "schedulerLastRunKey", schedulerLastRunKey)
}

// --- checker options ---

func TestCheckerOptionsPrimaryKeySize(t *testing.T) {
	// All three identifiers present; checker name can be arbitrary length.
	key := checkerOptionsKey("a-very-long-checker-plugin-name.v99", maxIDPtr, maxIDPtr, maxIDPtr)
	assertKeySize(t, "checkerOptionsPrimaryKey", key)
}

func TestCheckerOptionsNameIndexKeySize(t *testing.T) {
	compoundHash := hash28(checkerOptionsCompound("a-very-long-checker-plugin-name.v99", maxIDPtr, maxIDPtr, maxIDPtr))
	key := checkerOptionNameIndexKey("a-very-long-checker-plugin-name.v99", compoundHash)
	assertKeySize(t, "checkerOptionNameIndexKey", key)
}
