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
// As a counterpart to the access to the provider code and rights to copy, modify
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
	"fmt"
	"strings"

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/StackExchange/dnscontrol/v4/providers"
	"github.com/miekg/dns"
)

// Provider is where Domains and Zones can be managed.
type Provider interface {
	NewDNSServiceProvider() (providers.DNSServiceProvider, error)
	DNSControlName() string
}

// ProviderMeta holds the metadata associated to a Provider.
type ProviderMeta struct {
	// Type is the string representation of the Provider's type.
	Type string `json:"_srctype"`

	// Id is the Provider's identifier.
	Id Identifier `json:"_id"`

	// OwnerId is the User's identifier for the current Provider.
	OwnerId Identifier `json:"_ownerid"`

	// Comment is a string that helps user to distinguish the Provider.
	Comment string `json:"_comment,omitempty"`
}

// ProviderCombined combined ProviderMeta + Provider
type ProviderCombined struct {
	Provider
	ProviderMeta
}

func (p *ProviderCombined) Validate() (err error) {
	_, err = p.NewDNSServiceProvider()
	return
}

func (p *ProviderCombined) getZoneRecords(fqdn string) (rcs models.Records, err error) {
	var s providers.DNSServiceProvider
	s, err = p.NewDNSServiceProvider()
	if err != nil {
		return
	}

	defer func() {
		if a := recover(); a != nil {
			err = fmt.Errorf("%s", a)
		}
	}()

	return s.GetZoneRecords(strings.TrimSuffix(fqdn, "."), nil)
}

func (p *ProviderCombined) DomainExists(fqdn string) (err error) {
	_, err = p.getZoneRecords(fqdn)
	if err != nil {
		return
	}

	return nil
}

func (p *ProviderCombined) ImportZone(dn *Domain) (rrs []dns.RR, err error) {
	rcs, err := p.getZoneRecords(dn.DomainName)
	if err != nil {
		return rrs, err
	}

	for _, rc := range rcs {
		rrs = append(rrs, rc.ToRR())
	}

	return
}

func (p *ProviderCombined) GetDomainCorrections(dn *Domain, dc *models.DomainConfig) (rrs []*models.Correction, err error) {
	var s providers.DNSServiceProvider
	s, err = p.NewDNSServiceProvider()
	if err != nil {
		return
	}

	defer func() {
		if a := recover(); a != nil {
			err = fmt.Errorf("%s", a)
		}
	}()

	rcs, err := p.getZoneRecords(dn.DomainName)

	return s.GetZoneRecordsCorrections(dc, rcs)
}
