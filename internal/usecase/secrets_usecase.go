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
	"context"
	"fmt"
	"sort"

	"git.happydns.org/happyDomain/internal/secret"
)

// SecretHolder is a type of object holding secrets.
type SecretHolder interface {
	// InspectSecrets tells how the secrets of every object are stored.
	InspectSecrets(ctx context.Context) (secret.Counts, error)

	// ResealSecrets stores the secrets of every object the way the current
	// policy stores new ones.
	ResealSecrets(ctx context.Context) (secret.ResealReport, error)
}

// SecretsStatus tells how the secrets of the instance are stored.
type SecretsStatus struct {
	// Policy is how new secrets are stored.
	Policy string `json:"policy"`

	// Objects tells how the secrets are stored, by type of object.
	Objects map[string]secret.Counts `json:"objects"`
}

// SecretsUsecase lets the administrator follow and migrate how secrets are
// stored.
type SecretsUsecase struct {
	manager *secret.Manager
	holders map[string]SecretHolder
	checks  secret.CheckStorage
}

func NewSecretsUsecase(manager *secret.Manager, holders map[string]SecretHolder, checks secret.CheckStorage) *SecretsUsecase {
	return &SecretsUsecase{manager: manager, holders: holders, checks: checks}
}

func (u *SecretsUsecase) types() []string {
	types := make([]string, 0, len(u.holders))
	for t := range u.holders {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

// Status tells how the secrets of every object are stored.
func (u *SecretsUsecase) Status(ctx context.Context) (*SecretsStatus, error) {
	status := &SecretsStatus{
		Policy:  string(u.manager.Policy()),
		Objects: map[string]secret.Counts{},
	}

	for _, t := range u.types() {
		counts, err := u.holders[t].InspectSecrets(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", t, err)
		}
		status.Objects[t] = counts
	}

	return status, nil
}

// Reseal stores the secrets of every object the way the current policy
// stores new ones.
func (u *SecretsUsecase) Reseal(ctx context.Context) ([]secret.ResealReport, error) {
	var reports []secret.ResealReport

	for _, t := range u.types() {
		report, err := u.holders[t].ResealSecrets(ctx)
		if err != nil {
			return reports, fmt.Errorf("%s: %w", t, err)
		}
		reports = append(reports, report)
	}

	return reports, nil
}

// DropSafes deletes the safes and the keyset check record once every secret
// is stored in clear, after going back to the plaintext policy: the keyset is
// then no longer needed. It returns how many safes it deleted.
//
// It refuses while a secret is still sealed, opening or not, and while an
// object could not be looked at: it may hold sealed secrets, which would be
// lost for good.
func (u *SecretsUsecase) DropSafes(ctx context.Context) (int, error) {
	if u.manager.Policy() != secret.PolicyPlaintext {
		return 0, fmt.Errorf("switch to the %q secret policy, restart, and reseal first", secret.PolicyPlaintext)
	}

	status, err := u.Status(ctx)
	if err != nil {
		return 0, err
	}
	for _, t := range u.types() {
		c := status.Objects[t]
		sealed := c.Unreadable
		for _, n := range c.Sealed {
			sealed += n
		}
		if sealed > 0 {
			return 0, fmt.Errorf("%s: %d secrets are still sealed: reseal first, or have the unreadable ones entered again", t, sealed)
		}
		if c.Undecodable > 0 {
			return 0, fmt.Errorf("%s: %d objects could not be looked at and may hold sealed secrets: repair or delete them first", t, c.Undecodable)
		}
	}

	return u.manager.DropSafes(ctx, u.checks)
}
