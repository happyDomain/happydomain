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
)

type ZoneMeta struct {
	Id int64 `json:"id"`
}

type Zone struct {
	Id           int64                         `json:"id"`
	IdAuthor     int64                         `json:"id_author"`
	DefaultTTL   uint32                        `json:"default_ttl"`
	LastModified *time.Time                    `json:"last_modified,omitempty"`
	CommitMsg    *string                       `json:"commit_message,omitempty"`
	CommitDate   *time.Time                    `json:"commit_date,omitempty"`
	Published    *time.Time                    `json:"published,omitempty"`
	Aliases      map[string][]string           `json:"aliases"`
	Services     map[string][]*ServiceCombined `json:"services"`
}

func (z *Zone) FindService(id []byte) *ServiceCombined {
	for _, services := range z.Services {
		for _, svc := range services {
			if bytes.Equal(svc.Id, id) {
				return svc
			}
		}
	}

	return nil
}

func (z *Zone) EraseService(domain string, id []byte, s *ServiceCombined) error {
	if services, ok := z.Services[domain]; ok {
		for k, svc := range services {
			if bytes.Equal(svc.Id, id) {
				z.Services[domain][k] = s
				return nil
			}
		}
	}

	return errors.New("Service not found")
}
