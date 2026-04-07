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

package checkers

import (
	"context"
	"encoding/json"

	"git.happydns.org/happyDomain/model"
)

// stubObservationGetter is a test helper that serves a single pre-built
// observation, mimicking the SDK's mapObservationGetter (JSON round-trip).
type stubObservationGetter struct {
	key  happydns.ObservationKey
	data any
	err  error
}

func (g *stubObservationGetter) Get(ctx context.Context, key happydns.ObservationKey, dest any) error {
	if g.err != nil {
		return g.err
	}
	if key != g.key {
		return errNotFound
	}
	raw, err := json.Marshal(g.data)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dest)
}

type errString string

func (e errString) Error() string { return string(e) }

const errNotFound = errString("observation not available")

func newWhoisObs(d *WHOISData) *stubObservationGetter {
	return &stubObservationGetter{key: ObservationKeyWhois, data: d}
}
