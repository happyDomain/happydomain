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

// Package session provides the business logic for managing user sessions in
// happyDomain. It exposes a [Service] that handles the full session lifecycle:
// creation, retrieval, update, and deletion, as well as bulk operations such as
// closing all sessions for a given user.
//
// The package defines the [SessionStorage] interface that any persistence
// backend must implement. A concrete implementation is injected at construction
// time via [NewService], keeping this layer free of storage concerns.
//
// Session tokens are randomly generated, base32-encoded strings (see
// [NewSessionID]). A token is the credential the client presents and is never
// stored: sessions are stored and named by their public identifier,
// [happydns.SessionIDFromToken] of the token. Sessions carry an expiry timestamp and are automatically
// bound to a single user — cross-user access is rejected at the use-case level.
//
// Typical usage:
//
//	svc := session.NewService(myStorageBackend)
//
//	created, err := svc.CreateUserSession(user, "backup script")
//	// … hand created.Token to the user, it cannot be retrieved later …
//
//	sess, err := svc.GetUserSession(user, created.Id)
//
//	err = svc.UpdateUserSession(user, sessionID, func(s *happydns.Session) {
//	    s.Description = "renamed"
//	})
//
//	err = svc.DeleteUserSession(user, sessionID)
//
//	err = svc.CloseUserSessions(user) // invalidate all sessions at once
package session
