// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
)

// ParseRecords parses given text to retrieve inner records.
func (s *Server) ParseRecords(ctx context.Context, request ParseRecordsRequestObject) (ParseRecordsResponseObject, error) {
	origin := ""
	if request.Params.Origin != nil {
		origin = *request.Params.Origin
	}

	rrs, err := helpers.ParseRecord(*request.Body, origin)
	if err != nil {
		return ParseRecords400JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Failed to parse records: %s", err.Error()),
		}), nil
	}

	// Convert Record to map[string]interface{}
	var result map[string]interface{}
	data, err := json.Marshal(rrs)
	if err != nil {
		return ParseRecords400JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Failed to serialize record: %s", err.Error()),
		}), nil
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return ParseRecords400JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Failed to deserialize record: %s", err.Error()),
		}), nil
	}

	return ParseRecords200JSONResponse(result), nil
}
