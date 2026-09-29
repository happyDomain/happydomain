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
	"maps"
	"slices"
	"sort"

	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/model"
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

	// InstanceKeys lists the keys of the instance keyset and the number of
	// safes each wraps.
	InstanceKeys []secret.KeyStatus `json:"instanceKeys"`
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

// Status tells how the secrets of every object are stored, and which key of
// the instance keyset wraps how many safes.
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

	keys, err := u.manager.KeyStatus()
	if err != nil {
		return nil, err
	}
	status.InstanceKeys = keys

	return status, nil
}

// Rewrap wraps the key of every safe under the primary instance key, and
// reports what it did.
func (u *SecretsUsecase) Rewrap(ctx context.Context) (secret.ResealReport, error) {
	return u.manager.Rewrap(ctx)
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

// DropSafes deletes the instance safes once back to clear, so that the
// instance keyset can be removed from the configuration, and reports what it
// did. It refuses while a value still opens with them: run Reseal
// first. It also refuses while an object could not be inspected, as it may
// hold such values. Values that do not open any more do not stop it, as they
// are lost already.
func (u *SecretsUsecase) DropSafes(ctx context.Context) (secret.ResealReport, error) {
	none := secret.ResealReport{ObjectType: secret.SafeObjectType}
	if u.manager.Policy() != secret.PolicyPlaintext {
		return none, happydns.ValidationError{Msg: "switch to the plaintext secret policy first"}
	}

	status, err := u.Status(ctx)
	if err != nil {
		return none, err
	}
	// In order, so that the refusal names the same type every time.
	for _, t := range u.types() {
		counts := status.Objects[t]
		for _, kind := range slices.Sorted(maps.Keys(counts.Sealed)) {
			if n := counts.Sealed[kind]; n > 0 {
				return none, happydns.ValidationError{Msg: fmt.Sprintf("%d %s secret(s) still sealed: reseal first", n, t)}
			}
		}
		// What could not be looked at may hold values sealed in them.
		if counts.Undecodable > 0 {
			return none, happydns.ValidationError{Msg: fmt.Sprintf("%d %s object(s) could not be inspected and may still hold sealed secrets: retry, or repair or delete them, first (see the problems in the status)", counts.Undecodable, t)}
		}
	}

	return u.manager.DropInstanceSafes(ctx, u.checks)
}
