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

import "context"

type (
	keystoreKey    struct{}
	unlockScopeKey struct{}
)

// WithKeystore returns ctx carrying ref, the keystore the request chose for
// the secrets of the object it writes: see Destination.Keystore.
func WithKeystore(ctx context.Context, ref string) context.Context {
	if ref == "" {
		return ctx
	}
	return context.WithValue(ctx, keystoreKey{}, ref)
}

// KeystoreFromContext returns the keystore the request carried by ctx chose,
// empty when it chose none.
func KeystoreFromContext(ctx context.Context) string {
	ref, _ := ctx.Value(keystoreKey{}).(string)
	return ref
}

// WithUnlockScope returns ctx carrying scope, the session of the request: the
// keystores unlocked during it open for the requests carrying the same scope.
func WithUnlockScope(ctx context.Context, scope string) context.Context {
	if scope == "" {
		return ctx
	}
	return context.WithValue(ctx, unlockScopeKey{}, scope)
}

// UnlockScopeFromContext returns the scope ctx carries, empty for none, as in
// the background.
func UnlockScopeFromContext(ctx context.Context) string {
	scope, _ := ctx.Value(unlockScopeKey{}).(string)
	return scope
}
