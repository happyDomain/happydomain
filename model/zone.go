// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydns.org
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

	"github.com/miekg/dns"
)

type ZoneMeta struct {
	Id           int64      `json:"id"`
	IdAuthor     int64      `json:"id_author"`
	DefaultTTL   uint32     `json:"default_ttl"`
	LastModified time.Time  `json:"last_modified,omitempty"`
	CommitMsg    *string    `json:"commit_message,omitempty"`
	CommitDate   *time.Time `json:"commit_date,omitempty"`
	Published    *time.Time `json:"published,omitempty"`
}

type Zone struct {
	ZoneMeta
	Services map[string][]*ServiceCombined `json:"services"`
}

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

func (z *Zone) FindService(id []byte) (string, *ServiceCombined) {
	for subdomain := range z.Services {
		if svc := z.FindSubdomainService(subdomain, id); svc != nil {
			return subdomain, svc
		}
	}

	return "", nil
}

func (z *Zone) FindSubdomainService(domain string, id []byte) *ServiceCombined {
	for _, svc := range z.Services[domain] {
		if bytes.Equal(svc.Id, id) {
			return svc
		}
	}

	return nil
}

func (z *Zone) EraseService(subdomain string, origin string, id []byte, s *ServiceCombined) error {
	if services, ok := z.Services[subdomain]; ok {
		for k, svc := range services {
			if bytes.Equal(svc.Id, id) {
				if s == nil {
					// Disallow removing SOA
					if subdomain == "" && svc.Type == "svcs.Origin" {
						return errors.New("You cannot delete this service. It is mandatory.")
					}
					if len(z.Services[subdomain]) <= 1 {
						delete(z.Services, subdomain)
					} else {
						z.Services[subdomain] = append(z.Services[subdomain][:k], z.Services[subdomain][k+1:]...)
					}
				} else {
					s.Comment = s.GenComment(origin)
					s.NbResources = s.GetNbResources()
					z.Services[subdomain][k] = s
				}
				return nil
			}
		}
	}

	return errors.New("Service not found")
}

func (z *Zone) GenerateRRs(origin string) (rrs []dns.RR) {
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
	}

	return
}