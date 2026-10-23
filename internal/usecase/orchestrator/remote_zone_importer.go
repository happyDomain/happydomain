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

package orchestrator

import (
	"context"
	"fmt"
	"log"
	"maps"
	"runtime/debug"
	"slices"
	"time"

	"golang.org/x/sync/singleflight"

	"git.happydns.org/happyDomain/internal/helpers"
	domainlogUC "git.happydns.org/happyDomain/internal/usecase/domain_log"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/model"
)

// RemoteZoneImporterUsecase fetches the live DNS records for a domain directly
// from the provider and delegates to ZoneImporterUsecase to persist them.  It
// also appends a domain log entry on success.
type RemoteZoneImporterUsecase struct {
	appendDomainLog   domainlogUC.DomainLogAppender
	providerService   ProviderGetter
	zoneImporter      happydns.ZoneImporterUsecase
	zoneGetter        *zoneUC.GetZoneUsecase
	zoneRetriever     ZoneRetriever
	schedulerNotifier happydns.SchedulerDomainNotifier

	// inflight runs a single import at a time for each domain.
	inflight singleflight.Group
}

// remoteImportTimeout bounds an import, which no longer depends on the
// request that started it.
const remoteImportTimeout = 5 * time.Minute

// NewRemoteZoneImporterUsecase creates a RemoteZoneImporterUsecase wired to
// the given log appender, provider getter, zone importer, zone getter, and
// zone retriever.
func NewRemoteZoneImporterUsecase(
	appendDomainLog domainlogUC.DomainLogAppender,
	providerService ProviderGetter,
	zoneImporter happydns.ZoneImporterUsecase,
	zoneGetter *zoneUC.GetZoneUsecase,
	zoneRetriever ZoneRetriever,
) *RemoteZoneImporterUsecase {
	return &RemoteZoneImporterUsecase{
		appendDomainLog: appendDomainLog,
		providerService: providerService,
		zoneImporter:    zoneImporter,
		zoneGetter:      zoneGetter,
		zoneRetriever:   zoneRetriever,
	}
}

// Import resolves the provider for the domain, retrieves its current records,
// and imports them via ZoneImporterUsecase.  A domain log entry is appended on
// success.  Returns the newly created zone or an error.
//
// A single import runs at a time for a domain: a caller arriving while one
// is in flight gets its zone. The import outlives the callers: retrieving a
// zone may take long, and one leaving before the end would otherwise lose
// it, and start it over on coming back.
func (uc *RemoteZoneImporterUsecase) Import(ctx context.Context, user *happydns.User, domain *happydns.Domain) (*happydns.Zone, error) {
	done := uc.inflight.DoChan(domain.Id.String(), func() (zone any, err error) {
		// The import no longer runs in the request, out of reach of its
		// recovery: singleflight would raise a panic again where nothing
		// can catch it, bringing the whole server down.
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic while importing the zone of %s: %v\n%s", domain.DomainName, r, debug.Stack())
				zone, err = nil, fmt.Errorf("unable to import the zone of %s: internal error", domain.DomainName)
			}
		}()

		detached, cancel := context.WithTimeout(context.WithoutCancel(ctx), remoteImportTimeout)
		defer cancel()

		return uc.importNow(detached, user, domain)
	})

	select {
	case res := <-done:
		if res.Err != nil {
			return nil, res.Err
		}
		return res.Val.(*happydns.Zone), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// importNow retrieves the zone of domain from its provider and imports it.
func (uc *RemoteZoneImporterUsecase) importNow(ctx context.Context, user *happydns.User, domain *happydns.Domain) (*happydns.Zone, error) {
	provider, err := uc.providerService.GetUserProvider(ctx, user, domain.ProviderId)
	if err != nil {
		return nil, err
	}

	zone, err := uc.zoneRetriever.RetrieveZone(ctx, provider, domain.DomainName, uc.knownSubdomains(domain)...)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve the zone from server: %w", err)
	}

	myZone, err := uc.zoneImporter.Import(user, domain, zone)
	if err != nil {
		return nil, err
	}

	if err := uc.appendDomainLog.AppendDomainLog(domain, happydns.NewDomainLog(user, happydns.LOG_INFO, fmt.Sprintf("Zone imported from provider API: %s", myZone.Id.String()))); err != nil {
		log.Printf("unable to append domain log for %s: %s", domain.DomainName, err.Error())
	}

	if uc.schedulerNotifier != nil {
		uc.schedulerNotifier.NotifyDomainChange(domain)
	}

	return myZone, nil
}

// knownSubdomains returns the subdomains of the newest zone of domain, for
// the providers discovering the zone rather than listing it.
func (uc *RemoteZoneImporterUsecase) knownSubdomains(domain *happydns.Domain) []string {
	if len(domain.ZoneHistory) == 0 || uc.zoneGetter == nil {
		return nil
	}

	zone, err := uc.zoneGetter.Get(domain.ZoneHistory[0])
	if err != nil {
		log.Printf("unable to load the zone %s of %s to tell its known subdomains: %s", domain.ZoneHistory[0].String(), domain.DomainName, err)
		return nil
	}

	return subdomainsOf(zone)
}

// subdomainsOf returns the subdomains holding services in zone, and those
// owning their records: a service hangs from a parent of the names it
// fills, as a DKIM key or a DMARC policy from the domain they are about.
func subdomainsOf(zone *happydns.Zone) []string {
	seen := map[string]bool{}
	for subdomain, services := range zone.Services {
		seen[string(subdomain)] = true
		for _, svc := range services {
			records, err := svc.Service.GetRecords(string(subdomain), zone.DefaultTTL, "")
			if err != nil {
				continue
			}
			for _, rr := range records {
				seen[helpers.DomainJoin(rr.Header().Name, string(subdomain))] = true
			}
		}
	}
	return slices.Collect(maps.Keys(seen))
}
