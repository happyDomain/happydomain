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
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"

	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/services"
	"git.happydns.org/happydns/sources"
	"git.happydns.org/happydns/storage"
)

func init() {
	router.GET("/api/domains/:domain/zone/:zoneid", apiAuthHandler(domainHandler(zoneHandler(getZone))))
	router.PATCH("/api/domains/:domain/zone/:zoneid", apiAuthHandler(domainHandler(zoneHandler(updateZoneService))))

	router.GET("/api/domains/:domain/zone/:zoneid/:subdomain", apiAuthHandler(domainHandler(zoneHandler(getZoneSubdomain))))
	router.POST("/api/domains/:domain/zone/:zoneid/:subdomain/services", apiAuthHandler(domainHandler(zoneHandler(addZoneService))))
	router.GET("/api/domains/:domain/zone/:zoneid/:subdomain/services/:serviceid", apiAuthHandler(domainHandler(zoneHandler(getZoneService))))
	router.DELETE("/api/domains/:domain/zone/:zoneid/:subdomain/services/:serviceid", apiAuthHandler(domainHandler(zoneHandler(deleteZoneService))))
	router.GET("/api/domains/:domain/zone/:zoneid/:subdomain/services/:serviceid/records", apiAuthHandler(domainHandler(zoneHandler(getServiceRecords))))

	router.POST("/api/domains/:domain/import_zone", apiAuthHandler(domainHandler(importZone)))
	router.POST("/api/domains/:domain/view_zone/:zoneid", apiAuthHandler(domainHandler(zoneHandler(viewZone))))
	router.POST("/api/domains/:domain/apply_zone/:zoneid", apiAuthHandler(domainHandler(zoneHandler(applyZone))))
	router.POST("/api/domains/:domain/diff_zones/:zoneid1/:zoneid2", apiAuthHandler(domainHandler(diffZones)))
}

func zoneHandler(f func(*config.Options, *RequestResources, io.Reader) Response) func(*config.Options, *RequestResources, io.Reader) Response {
	return func(opts *config.Options, req *RequestResources, body io.Reader) Response {
		zoneid, err := strconv.ParseInt(req.Ps.ByName("zoneid"), 10, 64)
		if err != nil {
			return APIErrorResponse{
				err: err,
			}
		}

		// Check that the zoneid exists in the domain history
		if !req.Domain.HasZone(zoneid) {
			return APIErrorResponse{
				status: http.StatusNotFound,
				err:    errors.New("Zone not found"),
			}
		}

		if req.Zone, err = storage.MainStore.GetZone(zoneid); err != nil {
			return APIErrorResponse{
				status: http.StatusNotFound,
				err:    errors.New("Zone not found"),
			}
		} else {
			return f(opts, req, body)
		}
	}
}

func getZone(opts *config.Options, req *RequestResources, body io.Reader) Response {
	return APIResponse{
		response: req.Zone,
	}
}

func getZoneSubdomain(opts *config.Options, req *RequestResources, body io.Reader) Response {
	subdomain := strings.TrimSuffix(req.Ps.ByName("subdomain"), "@")
	return APIResponse{
		response: map[string]interface{}{
			"services": req.Zone.Services[subdomain],
		},
	}
}

func addZoneService(opts *config.Options, req *RequestResources, body io.Reader) Response {
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

	subdomain := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(req.Ps.ByName("subdomain"), "."+req.Domain.DomainName), "@"), req.Domain.DomainName)

	records := usc.Service.GenRRs(subdomain, usc.Ttl, req.Domain.DomainName)
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
	usc.Comment = usc.Service.GenComment(req.Domain.DomainName)

	req.Zone.Services[subdomain] = append(req.Zone.Services[subdomain], usc)

	err = storage.MainStore.UpdateZone(req.Zone)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	return APIResponse{
		response: req.Zone,
	}
}

func getZoneService(opts *config.Options, req *RequestResources, body io.Reader) Response {
	serviceid, err := hex.DecodeString(req.Ps.ByName("serviceid"))
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	return APIResponse{
		response: req.Zone.FindSubdomainService(req.Ps.ByName("subdomain"), serviceid),
	}
}

func importZone(opts *config.Options, req *RequestResources, body io.Reader) Response {
	source, err := storage.MainStore.GetSource(req.User, req.Domain.IdSource)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	zone, err := source.ImportZone(req.Domain)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	services, defaultTTL, err := svcs.AnalyzeZone(req.Domain.DomainName, zone)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	myZone := &happydns.Zone{
		ZoneMeta: happydns.ZoneMeta{
			IdAuthor:     req.Domain.IdUser,
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

	req.Domain.ZoneHistory = append(
		[]int64{myZone.Id}, req.Domain.ZoneHistory...)

	err = storage.MainStore.UpdateDomain(req.Domain)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	return APIResponse{
		response: &myZone.ZoneMeta,
	}
}

func diffZones(opts *config.Options, req *RequestResources, body io.Reader) Response {
	zoneid1, err := strconv.ParseInt(req.Ps.ByName("zoneid1"), 10, 64)
	if err != nil && req.Ps.ByName("zoneid1") != "@" {
		return APIErrorResponse{
			err: err,
		}
	}

	zoneid2, err := strconv.ParseInt(req.Ps.ByName("zoneid2"), 10, 64)
	if err != nil && req.Ps.ByName("zoneid2") != "@" {
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
		source, err := storage.MainStore.GetSource(req.User, req.Domain.IdSource)
		if err != nil {
			return APIErrorResponse{
				err: err,
			}
		}

		if zoneid1 == 0 {
			zone1, err = source.ImportZone(req.Domain)
		}
		if zoneid2 == 0 {
			zone2, err = source.ImportZone(req.Domain)
		}

		if err != nil {
			return APIErrorResponse{
				err: err,
			}
		}
	}

	if zoneid1 != 0 {
		if !req.Domain.HasZone(zoneid1) {
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
			zone1 = z1.GenerateRRs(req.Domain.DomainName)
		}
	}

	if zoneid2 != 0 {
		if !req.Domain.HasZone(zoneid2) {
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
			zone2 = z2.GenerateRRs(req.Domain.DomainName)
		}
	}

	toAdd, toDel := sources.DiffZones(zone1, zone2, true)

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

func applyZone(opts *config.Options, req *RequestResources, body io.Reader) Response {
	source, err := storage.MainStore.GetSource(req.User, req.Domain.IdSource)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	newSOA, err := sources.ApplyZone(source, req.Domain, req.Zone.GenerateRRs(req.Domain.DomainName), true)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	// Update serial
	if newSOA != nil {
		for _, svc := range req.Zone.Services[""] {
			if origin, ok := svc.Service.(*svcs.Origin); ok {
				origin.Serial = newSOA.Serial
				break
			}
		}
	}

	// Create a new zone in history for futher updates
	newZone := req.Zone.DerivateNew()
	//newZone.IdAuthor = //TODO get current user id
	err = storage.MainStore.CreateZone(newZone)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	req.Domain.ZoneHistory = append(
		[]int64{newZone.Id}, req.Domain.ZoneHistory...)

	err = storage.MainStore.UpdateDomain(req.Domain)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	// Commit changes in previous zone
	now := time.Now()
	// zone.ZoneMeta.IdAuthor = // TODO get current user id
	req.Zone.ZoneMeta.Published = &now

	req.Zone.LastModified = time.Now()

	err = storage.MainStore.UpdateZone(req.Zone)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	return APIResponse{
		response: newZone.ZoneMeta,
	}
}

func viewZone(opts *config.Options, req *RequestResources, body io.Reader) Response {
	var ret string

	for _, rr := range req.Zone.GenerateRRs(req.Domain.DomainName) {
		ret += rr.String() + "\n"
	}

	return APIResponse{
		response: ret,
	}
}

func updateZoneService(opts *config.Options, req *RequestResources, body io.Reader) Response {
	usc := &happydns.ServiceCombined{}
	err := json.NewDecoder(body).Decode(&usc)
	if err != nil {
		return APIErrorResponse{
			err: fmt.Errorf("Something is wrong in received data: %w", err),
		}
	}

	err = req.Zone.EraseService(usc.Domain, req.Domain.DomainName, usc.Id, usc)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	req.Zone.LastModified = time.Now()

	err = storage.MainStore.UpdateZone(req.Zone)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	return APIResponse{
		response: req.Zone,
	}
}

func deleteZoneService(opts *config.Options, req *RequestResources, body io.Reader) Response {
	serviceid, err := hex.DecodeString(req.Ps.ByName("serviceid"))
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	err = req.Zone.EraseService(req.Ps.ByName("subdomain"), req.Domain.DomainName, serviceid, nil)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	req.Zone.LastModified = time.Now()

	err = storage.MainStore.UpdateZone(req.Zone)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	return APIResponse{
		response: req.Zone,
	}
}

type serviceRecord struct {
	String string  `json:"string"`
	Fields *dns.RR `json:"fields,omitempty"`
}

func getServiceRecords(opts *config.Options, req *RequestResources, body io.Reader) Response {
	serviceid, err := hex.DecodeString(req.Ps.ByName("serviceid"))
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	svc := req.Zone.FindSubdomainService(req.Ps.ByName("subdomain"), serviceid)
	if svc == nil {
		return APIErrorResponse{
			err: errors.New("Service not found"),
		}
	}

	subdomain := req.Ps.ByName("subdomain")
	if subdomain == "" {
		subdomain = "@"
	}

	var ret []serviceRecord
	for _, rr := range svc.GenRRs(subdomain, 3600, req.Domain.DomainName) {
		ret = append(ret, serviceRecord{
			String: rr.String(),
			Fields: &rr,
		})
	}

	return APIResponse{
		response: ret,
	}
}
