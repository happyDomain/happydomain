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

package sources // import "happydns.org/sources"

import (
	"github.com/miekg/dns"

	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/utils"
)

func DiffZones(a []dns.RR, b []dns.RR, skipDNSSEC bool) (toAdd []dns.RR, toDel []dns.RR) {
loopDel:
	for _, rrA := range a {
		if skipDNSSEC && utils.IsDNSSECType(rrA.Header().Rrtype) {
			continue
		}
		for _, rrB := range b {
			if rrA.String() == rrB.String() {
				continue loopDel
			}
		}

		toDel = append(toDel, rrA)
	}

loopAdd:
	for _, rrB := range b {
		if skipDNSSEC && utils.IsDNSSECType(rrB.Header().Rrtype) {
			continue
		}
		for _, rrA := range a {
			if rrB.String() == rrA.String() {
				continue loopAdd
			}
		}

		toAdd = append(toAdd, rrB)
	}

	return
}

func DiffZone(s happydns.Source, domain *happydns.Domain, rrs []dns.RR, skipDNSSEC bool) (toAdd []dns.RR, toDel []dns.RR, err error) {
	// Get the actuals RR-set
	var current []dns.RR
	current, err = s.ImportZone(domain)
	if err != nil {
		return
	}

	toAdd, toDel = DiffZones(current, rrs, skipDNSSEC)
	return
}

func ApplyZone(s happydns.Source, domain *happydns.Domain, rrs []dns.RR, skipDNSSEC bool) (*dns.SOA, error) {
	toAdd, toDel, err := DiffZone(s, domain, rrs, skipDNSSEC)
	if err != nil {
		return nil, err
	}

	var newSOA *dns.SOA

	// Apply diff
	for _, rr := range toDel {
		if rr.Header().Rrtype == dns.TypeSOA {
			continue
		}
		if err := s.DeleteRR(domain, rr); err != nil {
			return nil, err
		}
	}
	for _, rr := range toAdd {
		if rr.Header().Rrtype == dns.TypeSOA {
			newSOA = rr.(*dns.SOA)
			continue
		}
		if err := s.AddRR(domain, rr); err != nil {
			return nil, err
		}
	}

	// Update SOA record
	if newSOA != nil {
		err = s.UpdateSOA(domain, newSOA, false)
	}

	return newSOA, err
}
