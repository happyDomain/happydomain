package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/miekg/dns"
)

func init() {
	router.GET("/api/zones/", apiHandler(getZones))
	router.GET("/api/zones/:zone/", apiHandler(zoneHandler(axfrZone)))
	router.PUT("/api/zones/:zone/", apiHandler(zoneHandler(addRR)))
	router.DELETE("/api/zones/:zone/", apiHandler(zoneHandler(delRR)))
}

func getZones(p httprouter.Params, body io.Reader) (Response) {
	return APIResponse{
		response: map[string][]string{
			"zones": []string{
				"adlin2020.p0m.fr.",
			},
		},
	}
}


func zoneHandler(f func(string, io.Reader) (Response)) func(httprouter.Params, io.Reader) (Response) {
	return func(ps httprouter.Params, body io.Reader) (Response) {
		zone := ps.ByName("zone")

		if zone[len(zone)-1] != '.' {
			return APIErrorResponse{
				err: errors.New("Not a valid full qualified domain name"),
			}
		}

		return f(zone, body)
	}
}

func axfrZone(zone string, body io.Reader) (Response) {
	t := new(dns.Transfer)

	m := new(dns.Msg)
	t.TsigSecret = map[string]string{"ddns.": "so6ZGir4GPAqINNh9U5c3A=="}
	m.SetAxfr(zone)
	m.SetTsig("ddns.", dns.HmacSHA256, 300, time.Now().Unix())

	c, err := t.In(m, "127.0.0.1:53")
	if err != nil {
		return APIErrorResponse{
			status: http.StatusInternalServerError,
			err: err,
		}
	}

	response := <-c
	var rrs []string

	for _, rr := range response.RR {
		rrs = append(rrs, rr.String())
	}

	if len(rrs) > 0 {
		rrs = rrs[0:len(rrs)-1]
	}

	return APIResponse{
		response: map[string][]string{
			"rr": rrs,
		},
	}
}

type uploadedRR struct {
	RR string `json:"rr"`
}

func addRR(zone string, body io.Reader) (Response) {
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

	m.Insert([]dns.RR{rr})

	c := new(dns.Client)
	c.TsigSecret = map[string]string{"ddns.": "so6ZGir4GPAqINNh9U5c3A=="}
	m.SetTsig("ddns.", dns.HmacSHA256, 300, time.Now().Unix())

	in, rtt, err := c.Exchange(m, "127.0.0.1:53")
	if err != nil {
		return APIErrorResponse{
			status: http.StatusInternalServerError,
			err: err,
		}
	}

	return APIResponse{
		response: map[string]interface{}{
			"in": *in,
			"rtt": rtt,
		},
	}
}

func delRR(zone string, body io.Reader) (Response) {
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

	in, rtt, err := c.Exchange(m, "127.0.0.1:53")
	if err != nil {
		return APIErrorResponse{
			status: http.StatusInternalServerError,
			err: err,
		}
	}

	return APIResponse{
		response: map[string]interface{}{
			"in": *in,
			"rtt": rtt,
		},
	}
}
