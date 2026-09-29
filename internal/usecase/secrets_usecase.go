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
}

func NewSecretsUsecase(manager *secret.Manager, holders map[string]SecretHolder) *SecretsUsecase {
	return &SecretsUsecase{manager: manager, holders: holders}
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
