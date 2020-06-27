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

package api

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/miekg/dns"

	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/services"
	"git.happydns.org/happydns/sources"
	"git.happydns.org/happydns/storage"
)

func init() {
	router.GET("/api/domains/:domain/zone/:zoneid", apiAuthHandler(zoneHandler(getZone)))
	router.PATCH("/api/domains/:domain/zone/:zoneid", apiAuthHandler(zoneHandler(updateZoneService)))

	router.GET("/api/domains/:domain/zone/:zoneid/:subdomain", apiAuthHandler(zoneHandler(getZoneSubdomain)))
	router.POST("/api/domains/:domain/zone/:zoneid/:subdomain", apiAuthHandler(zoneHandler(addZoneService)))
	router.GET("/api/domains/:domain/zone/:zoneid/:subdomain/*serviceid", apiAuthHandler(zoneHandler(getZoneService)))
	router.DELETE("/api/domains/:domain/zone/:zoneid/:subdomain/*serviceid", apiAuthHandler(zoneHandler(deleteZoneService)))

	router.POST("/api/domains/:domain/import_zone", apiAuthHandler(domainHandler(importZone)))
	router.POST("/api/domains/:domain/view_zone/:zoneid", apiAuthHandler(zoneHandler(viewZone)))
	router.POST("/api/domains/:domain/apply_zone/:zoneid", apiAuthHandler(zoneHandler(applyZone)))
	router.POST("/api/domains/:domain/diff_zones/:zoneid1/:zoneid2", apiAuthHandler(domainHandler(diffZones)))
}

func zoneHandler(f func(*config.Options, *happydns.Domain, *happydns.Zone, httprouter.Params, io.Reader) Response) func(*config.Options, *happydns.User, httprouter.Params, io.Reader) Response {
	return func(opts *config.Options, u *happydns.User, ps httprouter.Params, body io.Reader) Response {
		zoneid, err := strconv.ParseInt(ps.ByName("zoneid"), 10, 64)
		if err != nil {
			return APIErrorResponse{
				err: err,
			}
		}

		return domainHandler(func(opts *config.Options, domain *happydns.Domain, ps httprouter.Params, body io.Reader) Response {
			// Check that the zoneid exists in the domain history
			if !domain.HasZone(zoneid) {
				return APIErrorResponse{
					status: http.StatusNotFound,
					err:    errors.New("Zone not found"),
				}
			}

			if zone, err := storage.MainStore.GetZone(zoneid); err != nil {
				return APIErrorResponse{
					status: http.StatusNotFound,
					err:    errors.New("Zone not found"),
				}
			} else {
				return f(opts, domain, zone, ps, body)
			}
		})(opts, u, ps, body)
	}
}

func getZone(opts *config.Options, domain *happydns.Domain, zone *happydns.Zone, _ httprouter.Params, body io.Reader) Response {
	return APIResponse{
		response: zone,
	}
}

func getZoneSubdomain(opts *config.Options, domain *happydns.Domain, zone *happydns.Zone, ps httprouter.Params, body io.Reader) Response {
	subdomain := strings.TrimSuffix(ps.ByName("subdomain"), "@")
	return APIResponse{
		response: map[string]interface{}{
			"services": zone.Services[subdomain],
		},
	}
}

func addZoneService(opts *config.Options, domain *happydns.Domain, zone *happydns.Zone, ps httprouter.Params, body io.Reader) Response {
	usc := &happydns.ServiceCombined{}
	err := json.NewDecoder(body).Decode(&usc)
	if err != nil {
		return APIErrorResponse{
			err: fmt.Errorf("Something is wrong in received data: %w", err),
		}
	}

	if usc.Service == nil {
		return APIErrorResponse{
			err: fmt.Errorf("Unable to parse the given service."),
		}
	}

	subdomain := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(ps.ByName("subdomain"), "."+domain.DomainName), "@"), domain.DomainName)

	records := usc.Service.GenRRs(subdomain, usc.Ttl, domain.DomainName)
	if len(records) == 0 {
		return APIErrorResponse{
			err: fmt.Errorf("No record can be generated from your service."),
		}
	}

	hash := sha1.New()
	io.WriteString(hash, records[0].String())

	usc.Id = hash.Sum(nil)
	usc.Domain = subdomain
	usc.NbResources = usc.Service.GetNbResources()
	usc.Comment = usc.Service.GenComment(domain.DomainName)

	zone.Services[subdomain] = append(zone.Services[subdomain], usc)

	err = storage.MainStore.UpdateZone(zone)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	return APIResponse{
		response: zone,
	}
}

func getZoneService(opts *config.Options, domain *happydns.Domain, zone *happydns.Zone, ps httprouter.Params, body io.Reader) Response {
	serviceid, err := base64.StdEncoding.DecodeString(ps.ByName("serviceid")[1:])
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	return APIResponse{
		response: zone.FindSubdomainService(ps.ByName("subdomain"), serviceid),
	}
}

func importZone(opts *config.Options, domain *happydns.Domain, _ httprouter.Params, body io.Reader) Response {
	source, err := storage.MainStore.GetSource(&happydns.User{Id: domain.IdUser}, domain.IdSource)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	zone, err := source.ImportZone(domain)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	services, defaultTTL, err := svcs.AnalyzeZone(domain.DomainName, zone)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	myZone := &happydns.Zone{
		ZoneMeta: happydns.ZoneMeta{
			IdAuthor:     domain.IdUser,
			DefaultTTL:   defaultTTL,
			LastModified: time.Now(),
		},
		Services: services,
	}

	err = storage.MainStore.CreateZone(myZone)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	domain.ZoneHistory = append(
		[]int64{myZone.Id}, domain.ZoneHistory...)

	err = storage.MainStore.UpdateDomain(domain)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	return APIResponse{
		response: &myZone.ZoneMeta,
	}
}

func diffZones(opts *config.Options, domain *happydns.Domain, ps httprouter.Params, body io.Reader) Response {
	zoneid1, err := strconv.ParseInt(ps.ByName("zoneid1"), 10, 64)
	if err != nil && ps.ByName("zoneid1") != "@" {
		return APIErrorResponse{
			err: err,
		}
	}

	zoneid2, err := strconv.ParseInt(ps.ByName("zoneid2"), 10, 64)
	if err != nil && ps.ByName("zoneid2") != "@" {
		return APIErrorResponse{
			err: err,
		}
	}

	if zoneid1 == 0 && zoneid2 == 0 {
		return APIErrorResponse{
			err: fmt.Errorf("Both zoneId can't reference the live version"),
		}
	}

	var zone1 []dns.RR
	var zone2 []dns.RR

	if zoneid1 == 0 || zoneid2 == 0 {
		source, err := storage.MainStore.GetSource(&happydns.User{Id: domain.IdUser}, domain.IdSource)
		if err != nil {
			return APIErrorResponse{
				err: err,
			}
		}

		if zoneid1 == 0 {
			zone1, err = source.ImportZone(domain)
		}
		if zoneid2 == 0 {
			zone2, err = source.ImportZone(domain)
		}

		if err != nil {
			return APIErrorResponse{
				err: err,
			}
		}
	}

	if zoneid1 != 0 {
		if !domain.HasZone(zoneid1) {
			return APIErrorResponse{
				status: http.StatusNotFound,
				err:    errors.New("Zone A not found"),
			}
		} else if z1, err := storage.MainStore.GetZone(zoneid1); err != nil {
			return APIErrorResponse{
				status: http.StatusNotFound,
				err:    errors.New("Zone A not found"),
			}
		} else {
			zone1 = z1.GenerateRRs(domain.DomainName)
		}
	}

	if zoneid2 != 0 {
		if !domain.HasZone(zoneid2) {
			return APIErrorResponse{
				status: http.StatusNotFound,
				err:    errors.New("Zone B not found"),
			}
		} else if z2, err := storage.MainStore.GetZone(zoneid2); err != nil {
			return APIErrorResponse{
				status: http.StatusNotFound,
				err:    errors.New("Zone B not found"),
			}
		} else {
			zone2 = z2.GenerateRRs(domain.DomainName)
		}
	}

	toAdd, toDel := sources.DiffZones(zone1, zone2)

	var rrAdd []string
	for _, rr := range toAdd {
		rrAdd = append(rrAdd, rr.String())
	}

	var rrDel []string
	for _, rr := range toDel {
		rrDel = append(rrDel, rr.String())
	}

	return APIResponse{
		response: map[string]interface{}{
			"toAdd": rrAdd,
			"toDel": rrDel,
		},
	}
}

func applyZone(opts *config.Options, domain *happydns.Domain, zone *happydns.Zone, _ httprouter.Params, body io.Reader) Response {
	source, err := storage.MainStore.GetSource(&happydns.User{Id: domain.IdUser}, domain.IdSource)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	newSOA, err := sources.ApplyZone(source, domain, zone.GenerateRRs(domain.DomainName))
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	// Update serial
	if newSOA != nil {
		for _, svc := range zone.Services[""] {
			if origin, ok := svc.Service.(*svcs.Origin); ok {
				origin.Serial = newSOA.Serial
				break
			}
		}
	}

	// Create a new zone in history for futher updates
	newZone := zone.DerivateNew()
	//newZone.IdAuthor = //TODO get current user id
	err = storage.MainStore.CreateZone(newZone)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	domain.ZoneHistory = append(
		[]int64{newZone.Id}, domain.ZoneHistory...)

	err = storage.MainStore.UpdateDomain(domain)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	// Commit changes in previous zone
	now := time.Now()
	// zone.ZoneMeta.IdAuthor = // TODO get current user id
	zone.ZoneMeta.Published = &now

	zone.LastModified = time.Now()

	err = storage.MainStore.UpdateZone(zone)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	return APIResponse{
		response: newZone.ZoneMeta,
	}
}

func viewZone(opts *config.Options, domain *happydns.Domain, zone *happydns.Zone, _ httprouter.Params, body io.Reader) Response {
	var ret string

	for _, rr := range zone.GenerateRRs(domain.DomainName) {
		ret += rr.String() + "\n"
	}

	return APIResponse{
		response: ret,
	}
}

func updateZoneService(opts *config.Options, domain *happydns.Domain, zone *happydns.Zone, _ httprouter.Params, body io.Reader) Response {
	usc := &happydns.ServiceCombined{}
	err := json.NewDecoder(body).Decode(&usc)
	if err != nil {
		return APIErrorResponse{
			err: fmt.Errorf("Something is wrong in received data: %w", err),
		}
	}

	err = zone.EraseService(usc.Domain, domain.DomainName, usc.Id, usc)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	zone.LastModified = time.Now()

	err = storage.MainStore.UpdateZone(zone)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	return APIResponse{
		response: zone,
	}
}

func deleteZoneService(opts *config.Options, domain *happydns.Domain, zone *happydns.Zone, ps httprouter.Params, body io.Reader) Response {
	serviceid, err := base64.StdEncoding.DecodeString(ps.ByName("serviceid")[1:])
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	err = zone.EraseService(ps.ByName("subdomain"), domain.DomainName, serviceid, nil)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	zone.LastModified = time.Now()

	err = storage.MainStore.UpdateZone(zone)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	return APIResponse{
		response: zone,
	}
}
