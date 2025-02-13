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

package happydns

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/StackExchange/dnscontrol/v4/models"
)

// ZoneMeta holds the metadata associated to a Zone.
type ZoneMeta struct {
	// Id is the Zone's identifier.
	Id Identifier `json:"id" swaggertype:"string"`

	// IdAuthor is the User's identifier for the current Zone.
	IdAuthor Identifier `json:"id_author" swaggertype:"string"`

	// DefaultTTL is the TTL to use when no TTL has been defined for a record in this Zone.
	DefaultTTL uint32 `json:"default_ttl"`

	// LastModified holds the time when the last modification has been made on this Zone.
	LastModified time.Time `json:"last_modified,omitempty"`

	// CommitMsg is a message defined by the User to give a label to this Zone revision.
	CommitMsg *string `json:"commit_message,omitempty"`

	// CommitDate is the time when the commit has been made.
	CommitDate *time.Time `json:"commit_date,omitempty"`

	// Published indicates whether the Zone has already been published or not.
	Published *time.Time `json:"published,omitempty"`
}

// Zone contains ZoneMeta + map of services by subdomains.
type Zone struct {
	ZoneMeta
	Services map[string][]*ServiceCombined `json:"services"`
}

// DerivateNew creates a new Zone from the current one, by copying all fields.
func (z *Zone) DerivateNew() *Zone {
	newZone := new(Zone)

	newZone.ZoneMeta.IdAuthor = z.ZoneMeta.IdAuthor
	newZone.ZoneMeta.DefaultTTL = z.ZoneMeta.DefaultTTL
	newZone.ZoneMeta.LastModified = time.Now()
	newZone.Services = map[string][]*ServiceCombined{}

	for subdomain, svcs := range z.Services {
		newZone.Services[subdomain] = svcs
	}

	return newZone
}

func (z *Zone) AppendService(subdomain string, origin string, svc *ServiceCombined) error {
	hash, err := ValidateService(svc.Service, subdomain, origin)
	if err != nil {
		return err
	}

	svc.Id = hash
	svc.Domain = subdomain
	svc.NbResources = svc.Service.GetNbResources()
	svc.Comment = svc.Service.GenComment(origin)

	z.Services[subdomain] = append(z.Services[subdomain], svc)

	return nil
}

// FindService finds the Service identified by the given id.
func (z *Zone) FindService(id []byte) (string, *ServiceCombined) {
	for subdomain := range z.Services {
		if svc := z.FindSubdomainService(subdomain, id); svc != nil {
			return subdomain, svc
		}
	}

	return "", nil
}

func (z *Zone) findSubdomainService(subdomain string, id []byte) (int, *ServiceCombined) {
	if subdomain == "@" {
		subdomain = ""
	}

	if services, ok := z.Services[subdomain]; ok {
		for k, svc := range services {
			if bytes.Equal(svc.Id, id) {
				return k, svc
			}
		}
	}

	return -1, nil
}

// FindSubdomainService finds the Service identified by the given id, only under the given subdomain.
func (z *Zone) FindSubdomainService(domain string, id []byte) *ServiceCombined {
	_, svc := z.findSubdomainService(domain, id)
	return svc
}

func (z *Zone) eraseService(subdomain, origin string, old *ServiceCombined, idx int, new *ServiceCombined) error {
	if new == nil {
		// Disallow removing SOA
		if subdomain == "" && old.Type == "abstract.Origin" {
			return errors.New("You cannot delete this service. It is mandatory.")
		}

		if len(z.Services[subdomain]) <= 1 {
			delete(z.Services, subdomain)
		} else {
			z.Services[subdomain] = append(z.Services[subdomain][:idx], z.Services[subdomain][idx+1:]...)
		}
	} else {
		new.Comment = new.GenComment(origin)
		new.NbResources = new.GetNbResources()
		z.Services[subdomain][idx] = new
	}

	return nil
}

// EraseService overwrites the Service identified by the given id, under the given subdomain.
// The the new service is nil, it removes the existing Service instead of overwrite it.
func (z *Zone) EraseService(subdomain string, origin string, id []byte, s *ServiceCombined) error {
	if idx, svc := z.findSubdomainService(subdomain, id); svc != nil {
		return z.eraseService(subdomain, origin, svc, idx, s)
	}

	return errors.New("Service not found")
}

func (z *Zone) EraseServiceWithoutMeta(subdomain string, origin string, id []byte, s Service) error {
	if idx, svc := z.findSubdomainService(subdomain, id); svc != nil {
		return z.eraseService(subdomain, origin, svc, idx, &ServiceCombined{Service: s, ServiceMeta: svc.ServiceMeta})
	}

	return errors.New("Service not found")
}

// GenerateRRs returns all the reals records of the Zone.
func (z *Zone) GenerateRecords(origin string) (records models.Records, e error) {
	for subdomain, svcs := range z.Services {
		if subdomain == "" {
			subdomain = origin
		} else {
			subdomain += "." + origin
		}
		for _, svc := range svcs {
			var ttl uint32
			if svc.Ttl == 0 {
				ttl = z.DefaultTTL
			} else {
				ttl = svc.Ttl
			}

			rrs, err := svc.GetRecords(subdomain, ttl, origin)
			if err != nil {
				return nil, fmt.Errorf("unable to generate records for service %s: %w", svc, err)
			}

			for _, record := range rrs {
				if !strings.HasSuffix(record.Header().Name, ".") {
					if record.Header().Name == "" {
						record.Header().Name = subdomain
					} else {
						record.Header().Name += "." + subdomain
					}
				}

				rc, err := models.RRtoRC(record, strings.TrimSuffix(origin, "."))
				if err != nil {
					return nil, err
				}

				records = append(records, &rc)
			}
		}

		// Ensure SOA is the first record
		for i, rr := range records {
			if rr.Type == "SOA" {
				records[0], records[i] = records[i], records[0]
				break
			}
		}
	}

	return
}
