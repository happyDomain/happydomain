package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/miekg/dns"
)

func init() {
	router.GET("/api/zones", apiHandler(getZones))
	router.POST("/api/zones", apiHandler(addZone))
	router.DELETE("/api/zones/:zone", apiHandler(zoneHandler(delZone)))
	router.GET("/api/zones/:zone", apiHandler(zoneHandler(getZone)))
	router.GET("/api/zones/:zone/rr", apiHandler(zoneHandler(axfrZone)))
	router.POST("/api/zones/:zone/rr", apiHandler(zoneHandler(addRR)))
	router.DELETE("/api/zones/:zone/rr", apiHandler(zoneHandler(delRR)))
}

var tmpZones = []string{}

func getZones(p httprouter.Params, body io.Reader) Response {
	return APIResponse{
		response: tmpZones,
	}
}

func existsZone(zone string) bool {
	for _, z := range tmpZones {
		if z == zone {
			return true
		}
	}
	return false
}

type uploadedZone struct {
	Zone string `json:"domainName"`
}

func addZone(p httprouter.Params, body io.Reader) Response {
	var uz uploadedZone
	err := json.NewDecoder(body).Decode(&uz)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	if uz.Zone[len(uz.Zone)-1] != '.' {
		uz.Zone = uz.Zone + "."
	}

	if existsZone(uz.Zone) {
		return APIErrorResponse{
			err: errors.New("This zone already exists."),
		}
	} else {
		tmpZones = append(tmpZones, uz.Zone)
		return getZone(uz.Zone, body)
	}
}

func delZone(zone string, body io.Reader) Response {
	index := -1

	for i := range tmpZones {
		if tmpZones[i] == zone {
			index = i
			break
		}
	}

	if index == -1 {
		return APIErrorResponse{
			err: errors.New("This zone doesn't exist."),
		}
	}

	tmpZones = append(tmpZones[:index], tmpZones[index+1:]...)

	return APIResponse{
		response: true,
	}
}

func zoneHandler(f func(string, io.Reader) Response) func(httprouter.Params, io.Reader) Response {
	return func(ps httprouter.Params, body io.Reader) Response {
		zone := ps.ByName("zone")

		if !existsZone(zone) {
			return APIErrorResponse{
				status: http.StatusNotFound,
				err:    errors.New("Domain not found"),
			}
		}

		return f(zone, body)
	}
}

func getZone(zone string, body io.Reader) Response {
	return APIResponse{
		response: map[string]interface{}{
			"dn": zone,
		},
	}
}

func axfrZone(zone string, body io.Reader) Response {
	t := new(dns.Transfer)

	m := new(dns.Msg)
	t.TsigSecret = map[string]string{"ddns.": "so6ZGir4GPAqINNh9U5c3A=="}
	m.SetAxfr(zone)
	m.SetTsig("ddns.", dns.HmacSHA256, 300, time.Now().Unix())

	c, err := t.In(m, DefaultNameServer)
	if err != nil {
		return APIErrorResponse{
			status: http.StatusInternalServerError,
			err:    err,
		}
	}

	response := <-c
	var rrs []map[string]interface{}

	for _, rr := range response.RR {
		rrs = append(rrs, map[string]interface{}{
			"string": rr.String(),
			"fields": rr,
		})
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

func addRR(zone string, body io.Reader) Response {
	var urr uploadedRR
	err := json.NewDecoder(body).Decode(&urr)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	rr, err := dns.NewRR(fmt.Sprintf("$ORIGIN %s\n$TTL %d\n%s", zone, 3600, urr.RR))
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	m := new(dns.Msg)
	m.Id = dns.Id()
	m.Opcode = dns.OpcodeUpdate
	m.Question = make([]dns.Question, 1)
	m.Question[0] = dns.Question{zone, dns.TypeSOA, dns.ClassINET}

	m.Insert([]dns.RR{rr})

	c := new(dns.Client)
	c.TsigSecret = map[string]string{"ddns.": "so6ZGir4GPAqINNh9U5c3A=="}
	m.SetTsig("ddns.", dns.HmacSHA256, 300, time.Now().Unix())

	in, rtt, err := c.Exchange(m, DefaultNameServer)
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

func delRR(zone string, body io.Reader) Response {
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
	m.Question[0] = dns.Question{zone, dns.TypeSOA, dns.ClassINET}

	m.Remove([]dns.RR{rr})

	c := new(dns.Client)
	c.TsigSecret = map[string]string{"ddns.": "so6ZGir4GPAqINNh9U5c3A=="}
	m.SetTsig("ddns.", dns.HmacSHA256, 300, time.Now().Unix())

	in, rtt, err := c.Exchange(m, DefaultNameServer)
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
