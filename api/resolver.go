package api

import (
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/miekg/dns"

	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/model"
)

func init() {
	router.POST("/api/resolver", apiAuthHandler(runResolver))
}

type resolverRequest struct {
	Resolver   string `json:"resolver"`
	DomainName string `json:"domain"`
	Type       string `json:"type"`
}

func runResolver(_ *config.Options, u *happydns.User, _ httprouter.Params, body io.Reader) Response {
	var urr resolverRequest
	err := json.NewDecoder(body).Decode(&urr)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	client := dns.Client{Timeout: time.Second * 5}

	m := new(dns.Msg)
	m.SetQuestion(urr.DomainName, dns.StringToType[urr.Type])
	m.RecursionDesired = true
	m.SetEdns0(4096, true)

	r, _, err := client.Exchange(m, urr.Resolver+":53")
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	if r == nil {
		return APIErrorResponse{
			err: errors.New("response is nil"),
		}
	}
	if r.Rcode != dns.RcodeSuccess {
		return APIErrorResponse{
			err: errors.New("failed to get a valid answer"),
		}
	}

	return APIResponse{
		response: r.String(),
	}
}
