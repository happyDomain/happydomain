// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydomain.org
//
// This software is a computer program whose purpose is to provide a modern
// interface to interact with DNS systems.
//
// This software is governed by the CeCILL license under French law and abiding
// by the rules of distribution of free software.  You can use, modify and/or
// redistribute the software under the terms of the CeCILL license as
// circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and rights to copy, modify
// and redistribute granted by the license, users are provided only with a
// limited warranty and the software's author, the holder of the economic
// rights, and the successive licensors have only limited liability.
//
// In this respect, the user's attention is drawn to the risks associated with
// loading, using, modifying and/or developing or reproducing the software by
// the user in light of its specific status of free software, that may mean
// that it is complicated to manipulate, and that also therefore means that it
// is reserved for developers and experienced professionals having in-depth
// computer knowledge. Users are therefore encouraged to load and test the
// software's suitability as regards their requirements in conditions enabling
// the security of their systems and/or data to be ensured and, more generally,
// to use and operate it in the same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL license and that you accept its terms.

package happydns

import (
	"bytes"
	"errors"
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
func (z *Zone) GenerateRRs(origin string) (rrs models.Records) {
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
			rrs = append(rrs, svc.GenRRs(subdomain, ttl, origin)...)
		}

		// Ensure SOA is the first record
		for i, rr := range rrs {
			if rr.Type == "SOA" {
				rrs[0], rrs[i] = rrs[i], rrs[0]
				break
			}
		}
	}

	return
}
