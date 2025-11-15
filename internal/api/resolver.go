// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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

package api

import (
	"context"
	"encoding/json"
	"fmt"

	"git.happydns.org/happyDomain/model"
)

// ResolveDns performs a DNS resolution for a given domain, with options.
func (s *Server) ResolveDns(ctx context.Context, request ResolveDnsRequestObject) (ResolveDnsResponseObject, error) {
	r, err := s.dependancies.ResolverUsecase().ResolveQuestion(*request.Body)
	if err != nil {
		return ResolveDns500JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Failed to resolve: %s", err.Error()),
		}), nil
	}

	// Convert dns.Msg to happydns.DNSMsg
	var result happydns.DNSMsg
	data, err := json.Marshal(r)
	if err != nil {
		return ResolveDns500JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Failed to serialize DNS response: %s", err.Error()),
		}), nil
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return ResolveDns500JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Failed to deserialize DNS response: %s", err.Error()),
		}), nil
	}

	return ResolveDns200JSONResponse(result), nil
}
