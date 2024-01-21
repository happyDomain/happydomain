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

package actions

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/StackExchange/dnscontrol/v4/models"
	"go.uber.org/multierr"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
	"git.happydns.org/happyDomain/storage"
)

func DynamicUpdate(user *happydns.User, subdomains []string, ipv4, ipv6 string) error {
	domains, err := storage.MainStore.GetDomains(user)
	if err != nil {
		log.Printf("An error occurs when trying to GetDomains from DynamicUpdate: %s", err.Error())
		return fmt.Errorf("unable to retrive your domains.")
	}

	for _, hostname := range subdomains {
		var possibleDomains []string

		// Search domain name in account
		for _, dn := range domains {
			if strings.HasSuffix(hostname, dn.DomainName) {
				possibleDomains = append(possibleDomains, dn.DomainName)
			}
		}

		if len(possibleDomains) == 0 {
			return fmt.Errorf("Unable to find any parent domain for %q in your account. Please check you already registered a parent domain.", hostname)
		}

		// If many possibleDomains, find the most precise one
		if len(possibleDomains) > 1 {
			var domainWithMaxLen int
			var nbDomainWithMaxLen int = -1

			for i, dn := range possibleDomains {
				if len(possibleDomains[domainWithMaxLen]) == len(dn) {
					nbDomainWithMaxLen += 1
				} else if len(possibleDomains[domainWithMaxLen]) < len(dn) {
					domainWithMaxLen = i
					nbDomainWithMaxLen = 0
				}
			}

			// There are multiple domain with maximal precision, abort
			if nbDomainWithMaxLen > 1 {
				var dnList []string
				for _, dn := range possibleDomains {
					if len(dn) == len(possibleDomains[domainWithMaxLen]) {
						dnList = append(dnList, dn)
					}
				}

				return fmt.Errorf("Multiple registered domains in your account match for updating your given hostname: " + strings.Join(dnList, ", ") + ". I don't know which one use.")
			}

			possibleDomains = []string{possibleDomains[domainWithMaxLen]}
		}

		// dnscontrol wants hostname without leading .
		hostname = strings.TrimSuffix(hostname, ".")

		// Retrieve the domain ID
		var domainToUpdate *happydns.Domain

		for _, dn := range domains {
			if possibleDomains[0] == dn.DomainName {
				domainToUpdate = dn
				break
			}
		}

		// Retrieve corresponding provider
		provider, err := storage.MainStore.GetProvider(user, domainToUpdate.IdProvider)
		if err != nil {
			return fmt.Errorf("Unable to retrieve domain's provider: %w", err)
		}

		// Fetch the current zone
		zone, err := provider.ImportZone(domainToUpdate)
		if err != nil {
			return fmt.Errorf("Unable to retrieve current zone: %w", err)
		}

		// Make the modification
		var recordsToDrop []int
		for i, record := range zone {
			if record.GetLabelFQDN() == hostname {
				if (ipv4 != "" && record.Type == "A") ||
					(ipv6 != "" && record.Type == "AAAA") {
					recordsToDrop = append(recordsToDrop, i)
				}
			}
		}

		for i := len(recordsToDrop) - 1; i >= 0; i-- {
			zone = append(zone[0:recordsToDrop[i]], zone[recordsToDrop[i]+1:]...)
		}

		if ipv4 != "" {
			record := &models.RecordConfig{Type: "A"}
			record.SetLabelFromFQDN(hostname, strings.TrimSuffix(domainToUpdate.DomainName, "."))
			record.SetTarget(ipv4)

			zone = append(
				zone,
				record,
			)
		}
		if ipv6 != "" {
			record := &models.RecordConfig{Type: "AAAA"}
			record.SetLabelFromFQDN(hostname, strings.TrimSuffix(domainToUpdate.DomainName, "."))
			record.SetTarget(ipv6)

			zone = append(
				zone,
				record,
			)
		}

		// Push the new updated zone
		dc := &models.DomainConfig{
			Name:    strings.TrimSuffix(domainToUpdate.DomainName, "."),
			Records: zone,
		}

		corrections, err := provider.GetDomainCorrections(domainToUpdate, dc)
		if err != nil {
			return fmt.Errorf("Unable to compute domain corrections: %w", err)
		}

		var errs error
		for i, cr := range corrections {
			log.Printf("%s: apply ddns correction: %s", domainToUpdate.DomainName, cr.Msg)
			err := cr.F()
			if err != nil {
				log.Printf("%s: unable to apply ddns correction: %s", domainToUpdate.DomainName, err.Error())
				storage.MainStore.CreateDomainLog(domainToUpdate, happydns.NewDomainLog(user, happydns.LOG_ERR, fmt.Sprintf("DDNS API: Failed record update (%s): %s", cr.Msg, err.Error())))
				errs = multierr.Append(errs, fmt.Errorf("%s: %w", cr.Msg, err))

				// Stop the zone update if we didn't change it yet
				if i == 0 {
					break
				}
			}
		}

		if len(multierr.Errors(errs)) > 0 {
			return errs
		}

		// Prepare the corresponding history item
		services, defaultTTL, err := svcs.AnalyzeZone(domainToUpdate.DomainName, zone)
		if err != nil {
			return fmt.Errorf("Unable to perform the analysis of the new zone: %w", err)
		}

		now := time.Now()
		commitmsg := fmt.Sprintf("API DDNS update: IPv4=%s IPv6=%s", ipv4, ipv6)
		newZone := &happydns.Zone{
			ZoneMeta: happydns.ZoneMeta{
				IdAuthor:     domainToUpdate.IdUser,
				DefaultTTL:   defaultTTL,
				LastModified: now,
				CommitMsg:    &commitmsg,
				CommitDate:   &now,
				Published:    &now,
			},
			Services: services,
		}

		// Save in history
		err = storage.MainStore.CreateZone(newZone)
		if err != nil {
			return fmt.Errorf("unable to create the zone in history: %w", err)
		}

		storage.MainStore.CreateDomainLog(domainToUpdate, happydns.NewDomainLog(user, happydns.LOG_ACK, fmt.Sprintf("DDNS API: Zone published (%s), %d corrections applied with success", newZone.Id.String(), len(corrections))))

		if len(domainToUpdate.ZoneHistory) > 0 {
			domainToUpdate.ZoneHistory = append([]happydns.Identifier{domainToUpdate.ZoneHistory[0], newZone.Id}, domainToUpdate.ZoneHistory[1:]...)
		} else {
			domainToUpdate.ZoneHistory = []happydns.Identifier{newZone.Id}
		}

		err = storage.MainStore.UpdateDomain(domainToUpdate)
		if err != nil {
			return fmt.Errorf("unable to save the zone in history: %w", err)
		}
	}

	return nil
}
