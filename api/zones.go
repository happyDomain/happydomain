package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/miekg/dns"

	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/storage"
)

func init() {
	router.GET("/api/zones", apiAuthHandler(getZones))
	router.POST("/api/zones", apiAuthHandler(addZone))
	router.DELETE("/api/zones/:zone", apiAuthHandler(zoneHandler(delZone)))
	router.GET("/api/zones/:zone", apiAuthHandler(zoneHandler(getZone)))
	router.GET("/api/zones/:zone/rr", apiAuthHandler(zoneHandler(axfrZone)))
	router.POST("/api/zones/:zone/rr", apiAuthHandler(zoneHandler(addRR)))
	router.DELETE("/api/zones/:zone/rr", apiAuthHandler(zoneHandler(delRR)))
}

func getZones(_ *config.Options, u *happydns.User, p httprouter.Params, body io.Reader) Response {
	if zones, err := storage.MainStore.GetZones(u); err != nil {
		return APIErrorResponse{
			err: err,
		}
	} else {
		return APIResponse{
			response: zones,
		}
	}
}

func addZone(_ *config.Options, u *happydns.User, p httprouter.Params, body io.Reader) Response {
	var uz happydns.Zone
	err := json.NewDecoder(body).Decode(&uz)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	if len(uz.DomainName) <= 2 {
		return APIErrorResponse{
			err: errors.New("The given zone is invalid."),
		}
	}

	if uz.DomainName[len(uz.DomainName)-1] != '.' {
		uz.DomainName = uz.DomainName + "."
	}

	if len(uz.KeyName) > 1 && uz.KeyName[len(uz.KeyName)-1] != '.' {
		uz.KeyName = uz.KeyName + "."
	}

	if storage.MainStore.ZoneExists(uz.DomainName) {
		return APIErrorResponse{
			err: errors.New("This zone already exists."),
		}
	} else if err := storage.MainStore.CreateZone(u, &uz); err != nil {
		return APIErrorResponse{
			err: err,
		}
	} else {
		return APIResponse{
			response: uz,
		}
	}
}

func delZone(_ *config.Options, zone *happydns.Zone, body io.Reader) Response {
	if err := storage.MainStore.DeleteZone(zone); err != nil {
		return APIErrorResponse{
			err: err,
		}
	} else {
		return APIResponse{
			response: true,
		}
	}
}

func zoneHandler(f func(*config.Options, *happydns.Zone, io.Reader) Response) func(*config.Options, *happydns.User, httprouter.Params, io.Reader) Response {
	return func(opts *config.Options, u *happydns.User, ps httprouter.Params, body io.Reader) Response {
		if zone, err := storage.MainStore.GetZoneByDN(u, ps.ByName("zone")); err != nil {
			return APIErrorResponse{
				status: http.StatusNotFound,
				err:    errors.New("Domain not found"),
			}
		} else {
			return f(opts, zone, body)
		}
	}
}

func getZone(_ *config.Options, zone *happydns.Zone, body io.Reader) Response {
	return APIResponse{
		response: zone,
	}
}

func normalizeNSServer(opts *config.Options, srv string) string {
	if srv == "" {
		return opts.DefaultNameServer
	} else if strings.Index(srv, ":") > -1 {
		return srv
	} else {
		return srv + ":53"
	}
}

func axfrZone(opts *config.Options, zone *happydns.Zone, body io.Reader) Response {
	d := net.Dialer{}
	con, err := d.Dial("tcp", normalizeNSServer(opts, zone.Server))
	if err != nil {
		return APIErrorResponse{
			status: http.StatusInternalServerError,
			err:    err,
		}
	}
	defer con.Close()

	m := new(dns.Msg)
	m.SetEdns0(4096, true)
	m.SetAxfr(zone.DomainName)
	m.SetTsig(zone.KeyName, zone.KeyAlgo, 300, time.Now().Unix())

	dnscon := &dns.Conn{Conn: con}
	transfer := &dns.Transfer{Conn: dnscon, TsigSecret: map[string]string{zone.KeyName: zone.Base64KeyBlob()}}
	c, err := transfer.In(m, normalizeNSServer(opts, zone.Server))

	if err != nil {
		return APIErrorResponse{
			status: http.StatusInternalServerError,
			err:    err,
		}
	}

	var rrs []map[string]interface{}
	for {
		response, ok := <-c
		if !ok {
			break
		}

		for _, rr := range response.RR {
			rrs = append(rrs, map[string]interface{}{
				"string": rr.String(),
				"fields": rr,
			})
		}
	}

	if len(rrs) > 0 {
		rrs = rrs[0 : len(rrs)-1]
	}

	return APIResponse{
		response: rrs,
	}
}

type uploadedRR struct {
	RR string `json:"string"`
}

func addRR(opts *config.Options, zone *happydns.Zone, body io.Reader) Response {
	var urr uploadedRR
	err := json.NewDecoder(body).Decode(&urr)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	rr, err := dns.NewRR(fmt.Sprintf("$ORIGIN %s\n$TTL %d\n%s", zone.DomainName, 3600, urr.RR))
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	m := new(dns.Msg)
	m.Id = dns.Id()
	m.Opcode = dns.OpcodeUpdate
	m.Question = make([]dns.Question, 1)
	m.Question[0] = dns.Question{zone.DomainName, dns.TypeSOA, dns.ClassINET}

	m.Insert([]dns.RR{rr})

	c := new(dns.Client)
	c.TsigSecret = map[string]string{zone.KeyName: zone.Base64KeyBlob()}
	m.SetTsig(zone.KeyName, zone.KeyAlgo, 300, time.Now().Unix())

	in, rtt, err := c.Exchange(m, normalizeNSServer(opts, zone.Server))
	if err != nil {
		return APIErrorResponse{
			status: http.StatusInternalServerError,
			err:    err,
		}
	}

	return APIResponse{
		response: map[string]interface{}{
			"in":     *in,
			"rtt":    rtt,
			"string": rr.String(),
		},
	}
}

func delRR(opts *config.Options, zone *happydns.Zone, body io.Reader) Response {
	var urr uploadedRR
	err := json.NewDecoder(body).Decode(&urr)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	rr, err := dns.NewRR(urr.RR)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	m := new(dns.Msg)
	m.Id = dns.Id()
	m.Opcode = dns.OpcodeUpdate
	m.Question = make([]dns.Question, 1)
	m.Question[0] = dns.Question{zone.DomainName, dns.TypeSOA, dns.ClassINET}

	m.Remove([]dns.RR{rr})

	c := new(dns.Client)
	c.TsigSecret = map[string]string{zone.KeyName: zone.Base64KeyBlob()}
	m.SetTsig(zone.KeyName, zone.KeyAlgo, 300, time.Now().Unix())

	in, rtt, err := c.Exchange(m, normalizeNSServer(opts, zone.Server))
	if err != nil {
		return APIErrorResponse{
			status: http.StatusInternalServerError,
			err:    err,
		}
	}

	return APIResponse{
		response: map[string]interface{}{
			"in":  *in,
			"rtt": rtt,
		},
	}
}
