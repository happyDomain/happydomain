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
	"fmt"
	"strings"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
)

// GetZone retrieves the specified zone with its services and records.
func (s *Server) GetZone(ctx context.Context, request GetZoneRequestObject) (GetZoneResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return GetZone401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	_, zone, err := s.GetUserDomainAndZone(user, request.DomainId, request.ZoneId)
	if err != nil {
		return GetZone404JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	return GetZone200JSONResponse(*zone), nil
}

// UpdateService adds or updates a service inside the given Zone.
func (s *Server) UpdateService(ctx context.Context, request UpdateServiceRequestObject) (UpdateServiceResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return UpdateService401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	_, _, err = s.GetUserDomainAndZone(user, request.DomainId, request.ZoneId)
	if err != nil {
		return UpdateService404JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	// TODO: Parse service from request.Body
	// This needs serviceUC.ParseService which we'll need to adapt
	return UpdateService404JSONResponse(happydns.ErrorResponse{
		Message: "Not implemented yet",
	}), nil
}

// ListSubdomainServices returns the services associated with the given subdomain.
func (s *Server) ListSubdomainServices(ctx context.Context, request ListSubdomainServicesRequestObject) (ListSubdomainServicesResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return ListSubdomainServices401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	_, zone, err := s.GetUserDomainAndZone(user, request.DomainId, request.ZoneId)
	if err != nil {
		return ListSubdomainServices404JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	return ListSubdomainServices200JSONResponse{
		Services: zone.Services[request.Subdomain],
	}, nil
}

// RetrieveZone retrieves the current zone deployed on the NS Provider.
func (s *Server) RetrieveZone(ctx context.Context, request RetrieveZoneRequestObject) (RetrieveZoneResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return RetrieveZone401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	domain, err := s.GetUserDomainById(user, request.DomainId)
	if err != nil {
		return RetrieveZone404JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	zone, err := s.dependancies.RemoteZoneImporterUsecase().Import(user, domain)
	if err != nil {
		return RetrieveZone404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Failed to retrieve zone: %s", err.Error()),
		}), nil
	}

	return RetrieveZone200JSONResponse(zone.Meta()), nil
}

// AddService adds a Service to the given subdomain of the Zone.
func (s *Server) AddService(ctx context.Context, request AddServiceRequestObject) (AddServiceResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return AddService401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	_, _, err = s.GetUserDomainAndZone(user, request.DomainId, request.ZoneId)
	if err != nil {
		return AddService404JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	// TODO: Parse service from request body and add it to zone
	// This needs serviceUC.ParseService and AddServiceToZone
	return AddService404JSONResponse(happydns.ErrorResponse{
		Message: "Not implemented yet",
	}), nil
}

// GetService retrieves the designated Service.
func (s *Server) GetService(ctx context.Context, request GetServiceRequestObject) (GetServiceResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return GetService401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	_, zone, err := s.GetUserDomainAndZone(user, request.DomainId, request.ZoneId)
	if err != nil {
		return GetService404JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	serviceid, err := happydns.NewIdentifierFromString(request.ServiceId)
	if err != nil {
		return GetService404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Invalid service ID: %s", err.Error()),
		}), nil
	}

	_, svc := zone.FindSubdomainService(request.Subdomain, serviceid)
	if svc == nil {
		return GetService404JSONResponse(happydns.ErrorResponse{
			Message: "Service not found",
		}), nil
	}

	return GetService200JSONResponse(*svc), nil
}

// DeleteService drops the given Service.
func (s *Server) DeleteService(ctx context.Context, request DeleteServiceRequestObject) (DeleteServiceResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return DeleteService401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	domain, zone, err := s.GetUserDomainAndZone(user, request.DomainId, request.ZoneId)
	if err != nil {
		return DeleteService404JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	serviceid, err := happydns.NewIdentifierFromString(request.ServiceId)
	if err != nil {
		return DeleteService400JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Invalid service ID: %s", err.Error()),
		}), nil
	}

	zone, err = s.dependancies.ZoneServiceUsecase().RemoveServiceFromZone(user, domain, zone, request.Subdomain, serviceid)
	if err != nil {
		return DeleteService404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Failed to remove service: %s", err.Error()),
		}), nil
	}

	return DeleteService200JSONResponse(*zone), nil
}

// DiffZones computes the difference between the two zone identifiers given.
func (s *Server) DiffZones(ctx context.Context, request DiffZonesRequestObject) (DiffZonesResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return DiffZones401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	domain, newzone, err := s.GetUserDomainAndZone(user, request.DomainId, request.ZoneId)
	if err != nil {
		return DiffZones404JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	if request.OldZoneId == "@" {
		corrections, err := s.dependancies.ZoneCorrectionApplierUsecase().List(user, domain, newzone)
		if err != nil {
			return DiffZones500JSONResponse(happydns.ErrorResponse{
				Message: fmt.Sprintf("Failed to compute differences: %s", err.Error()),
			}), nil
		}
		var result []happydns.Correction
		for _, c := range corrections {
			result = append(result, c.Correction)
		}
		return DiffZones200JSONResponse(result), nil
	} else {
		oldzoneid, err := happydns.NewIdentifierFromString(request.OldZoneId)
		if err != nil {
			return DiffZones400JSONResponse(happydns.ErrorResponse{
				Message: fmt.Sprintf("Invalid old zone ID: %s", err.Error()),
			}), nil
		}

		corrections, err := s.dependancies.ZoneUsecase().DiffZones(domain, newzone, oldzoneid)
		if err != nil {
			return DiffZones500JSONResponse(happydns.ErrorResponse{
				Message: fmt.Sprintf("Failed to compute differences: %s", err.Error()),
			}), nil
		}
		var result []happydns.Correction
		for _, c := range corrections {
			result = append(result, *c)
		}
		return DiffZones200JSONResponse(result), nil
	}
}

// ApplyZoneChanges performs the requested changes with the provider.
func (s *Server) ApplyZoneChanges(ctx context.Context, request ApplyZoneChangesRequestObject) (ApplyZoneChangesResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return ApplyZoneChanges401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	domain, zone, err := s.GetUserDomainAndZone(user, request.DomainId, request.ZoneId)
	if err != nil {
		return ApplyZoneChanges404JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	// Convert []string to []Identifier for WantedCorrections
	var wantedCorrections []happydns.Identifier
	for _, id := range *request.Body {
		identifier, err := happydns.NewIdentifierFromString(id)
		if err != nil {
			return ApplyZoneChanges400JSONResponse(happydns.ErrorResponse{
				Message: fmt.Sprintf("Invalid correction ID: %s", err.Error()),
			}), nil
		}
		wantedCorrections = append(wantedCorrections, identifier)
	}

	form := happydns.ApplyZoneForm{
		WantedCorrections: wantedCorrections,
		CommitMsg:         "",
	}

	newZone, err := s.dependancies.ZoneCorrectionApplierUsecase().Apply(user, domain, zone, &form)
	if err != nil {
		return ApplyZoneChanges500JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Failed to apply changes: %s", err.Error()),
		}), nil
	}

	return ApplyZoneChanges200JSONResponse(newZone.Meta()), nil
}

// ViewZone creates a flatten export of the zone.
func (s *Server) ViewZone(ctx context.Context, request ViewZoneRequestObject) (ViewZoneResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return ViewZone401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	domain, zone, err := s.GetUserDomainAndZone(user, request.DomainId, request.ZoneId)
	if err != nil {
		return ViewZone404JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	ret, err := s.dependancies.ZoneUsecase().FlattenZoneFile(domain, zone)
	if err != nil {
		return ViewZone404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Failed to export zone: %s", err.Error()),
		}), nil
	}

	return ViewZone200JSONResponse(ret), nil
}

// AddZoneRecords adds given records in the zone.
func (s *Server) AddZoneRecords(ctx context.Context, request AddZoneRecordsRequestObject) (AddZoneRecordsResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return AddZoneRecords401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	domainId, err := happydns.NewIdentifierFromString(request.DomainId)
	if err != nil {
		return AddZoneRecords404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Invalid domain ID: %s", err.Error()),
		}), nil
	}

	zoneId, err := happydns.NewIdentifierFromString(request.ZoneId)
	if err != nil {
		return AddZoneRecords404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Invalid zone ID: %s", err.Error()),
		}), nil
	}

	domain, err := s.dependancies.DomainUsecase().GetUserDomain(user, domainId)
	if err != nil {
		return AddZoneRecords404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Domain not found: %s", err.Error()),
		}), nil
	}

	zone, err := s.dependancies.ZoneUsecase().GetZone(zoneId)
	if err != nil {
		return AddZoneRecords404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Zone not found: %s", err.Error()),
		}), nil
	}

	for _, record := range *request.Body {
		rr, err := helpers.ParseRecord(record, domain.Domain)
		if err != nil {
			return AddZoneRecords404JSONResponse(happydns.ErrorResponse{
				Message: fmt.Sprintf("Failed to parse record: %s", err.Error()),
			}), nil
		}

		// Make record relative
		rr = helpers.RRRelative(rr, domain.Domain)

		if strings.HasSuffix(rr.Header().Name, ".") {
			return AddZoneRecords404JSONResponse(happydns.ErrorResponse{
				Message: fmt.Sprintf("Record %q is not part of the current domain: %s", rr.Header().String(), domain.Domain),
			}), nil
		}

		err = s.dependancies.ZoneUsecase().AddRecord(zone, domain.Domain, rr)
		if err != nil {
			return AddZoneRecords404JSONResponse(happydns.ErrorResponse{
				Message: fmt.Sprintf("Failed to add record: %s", err.Error()),
			}), nil
		}
	}

	err = s.dependancies.ZoneUsecase().UpdateZone(zone.Id, func(z *happydns.Zone) {
		z.Services = zone.Services
	})
	if err != nil {
		return AddZoneRecords404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Failed to update zone: %s", err.Error()),
		}), nil
	}

	return AddZoneRecords200JSONResponse(*zone), nil
}

// DeleteZoneRecords deletes given records in the zone.
func (s *Server) DeleteZoneRecords(ctx context.Context, request DeleteZoneRecordsRequestObject) (DeleteZoneRecordsResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return DeleteZoneRecords401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	domainId, err := happydns.NewIdentifierFromString(request.DomainId)
	if err != nil {
		return DeleteZoneRecords404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Invalid domain ID: %s", err.Error()),
		}), nil
	}

	zoneId, err := happydns.NewIdentifierFromString(request.ZoneId)
	if err != nil {
		return DeleteZoneRecords404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Invalid zone ID: %s", err.Error()),
		}), nil
	}

	domain, err := s.dependancies.DomainUsecase().GetUserDomain(user, domainId)
	if err != nil {
		return DeleteZoneRecords404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Domain not found: %s", err.Error()),
		}), nil
	}

	zone, err := s.dependancies.ZoneUsecase().GetZone(zoneId)
	if err != nil {
		return DeleteZoneRecords404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Zone not found: %s", err.Error()),
		}), nil
	}

	for _, record := range *request.Body {
		rr, err := helpers.ParseRecord(record, domain.Domain)
		if err != nil {
			return DeleteZoneRecords404JSONResponse(happydns.ErrorResponse{
				Message: fmt.Sprintf("Failed to parse record: %s", err.Error()),
			}), nil
		}

		// Make record relative
		rr = helpers.RRRelative(rr, domain.Domain)

		err = s.dependancies.ZoneUsecase().DeleteRecord(zone, domain.Domain, rr)
		if err != nil {
			return DeleteZoneRecords404JSONResponse(happydns.ErrorResponse{
				Message: fmt.Sprintf("Failed to delete record: %s", err.Error()),
			}), nil
		}
	}

	err = s.dependancies.ZoneUsecase().UpdateZone(zone.Id, func(z *happydns.Zone) {
		z.Services = zone.Services
	})
	if err != nil {
		return DeleteZoneRecords404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Failed to update zone: %s", err.Error()),
		}), nil
	}

	return DeleteZoneRecords200JSONResponse(*zone), nil
}

// UpdateZoneRecord updates a given record in the zone.
func (s *Server) UpdateZoneRecord(ctx context.Context, request UpdateZoneRecordRequestObject) (UpdateZoneRecordResponseObject, error) {
	_, user, err := s.GetUserFromContext(ctx)
	if err != nil {
		return UpdateZoneRecord401JSONResponse(happydns.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	domainId, err := happydns.NewIdentifierFromString(request.DomainId)
	if err != nil {
		return UpdateZoneRecord404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Invalid domain ID: %s", err.Error()),
		}), nil
	}

	zoneId, err := happydns.NewIdentifierFromString(request.ZoneId)
	if err != nil {
		return UpdateZoneRecord404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Invalid zone ID: %s", err.Error()),
		}), nil
	}

	domain, err := s.dependancies.DomainUsecase().GetUserDomain(user, domainId)
	if err != nil {
		return UpdateZoneRecord404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Domain not found: %s", err.Error()),
		}), nil
	}

	zone, err := s.dependancies.ZoneUsecase().GetZone(zoneId)
	if err != nil {
		return UpdateZoneRecord404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Zone not found: %s", err.Error()),
		}), nil
	}

	oldRecord, err := helpers.ParseRecord(request.Body.OldRR, domain.Domain)
	if err != nil {
		return UpdateZoneRecord404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Failed to parse old record: %s", err.Error()),
		}), nil
	}

	newRecord, err := helpers.ParseRecord(request.Body.NewRR, domain.Domain)
	if err != nil {
		return UpdateZoneRecord404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Failed to parse new record: %s", err.Error()),
		}), nil
	}

	// Make records relative
	oldRecord = helpers.RRRelative(oldRecord, domain.Domain)
	newRecord = helpers.RRRelative(newRecord, domain.Domain)

	err = s.dependancies.ZoneUsecase().DeleteRecord(zone, domain.Domain, oldRecord)
	if err != nil {
		return UpdateZoneRecord404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Failed to delete old record: %s", err.Error()),
		}), nil
	}

	err = s.dependancies.ZoneUsecase().AddRecord(zone, domain.Domain, newRecord)
	if err != nil {
		return UpdateZoneRecord404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Failed to add new record: %s", err.Error()),
		}), nil
	}

	err = s.dependancies.ZoneUsecase().UpdateZone(zone.Id, func(z *happydns.Zone) {
		z.Services = zone.Services
	})
	if err != nil {
		return UpdateZoneRecord404JSONResponse(happydns.ErrorResponse{
			Message: fmt.Sprintf("Failed to update zone: %s", err.Error()),
		}), nil
	}

	return UpdateZoneRecord200JSONResponse(*zone), nil
}
