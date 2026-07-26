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

package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"git.happydns.org/happyDomain/internal/forms"
	"git.happydns.org/happyDomain/internal/netguard"
	"git.happydns.org/happyDomain/model"
)

// endpointCheckTimeout bounds the name resolution done before a provider is
// instantiated. It is short: this runs on the request path, and the answer is
// almost always already in the resolver's cache.
const endpointCheckTimeout = 5 * time.Second

// checkEndpoints refuses a provider whose configured destination is one this
// instance may not reach.
//
// It runs before the provider is instantiated, and that ordering is the
// security property: several DNSControl backends dial during construction, and
// none of them accepts a dialer we could wrap, so a check made afterwards would
// be made after the credentials have already left.
//
// The destination is re-resolved inside the backend, which leaves a rebinding
// window we cannot close from here. Narrowing it further would mean vendoring a
// dialer into every one of the 60+ backends.
func checkEndpoints(ctx context.Context, guard *netguard.Guard, body happydns.ProviderBody) error {
	endpoints := forms.Endpoints(body)
	if len(endpoints) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, endpointCheckTimeout)
	defer cancel()

	for _, endpoint := range endpoints {
		host, err := netguard.HostFromEndpoint(endpoint.Value)
		if err != nil {
			return happydns.ValidationError{
				Msg: fmt.Sprintf("field %q is not a usable endpoint: %s", endpoint.Label, err),
			}
		}

		if _, err := guard.ResolveAllowed(ctx, host); err != nil {
			subject := fmt.Sprintf("The %q field", endpoint.Label)

			// A resolver that timed out has not told us anything about this
			// endpoint. Refusing it as forbidden would send the user editing a
			// value that may be correct, and would hide a resolver outage
			// behind a configuration complaint.
			if errors.Is(err, netguard.ErrTemporary) {
				return happydns.InternalError{
					Err:         fmt.Errorf("unable to check endpoint %q: %w", endpoint.Label, err),
					UserMessage: guard.Unavailable(subject),
				}
			}

			return happydns.ValidationError{Msg: guard.Refusal(subject)}
		}
	}

	return nil
}
