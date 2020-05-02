package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/miekg/dns"

	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/storage"
)

func init() {
	router.GET("/api/domains", apiAuthHandler(getDomains))
	router.POST("/api/domains", apiAuthHandler(addDomain))
	router.DELETE("/api/domains/:domain", apiAuthHandler(domainHandler(delDomain)))
	router.GET("/api/domains/:domain", apiAuthHandler(domainHandler(getDomain)))
	router.GET("/api/domains/:domain/rr", apiAuthHandler(domainHandler(axfrDomain)))
	router.POST("/api/domains/:domain/rr", apiAuthHandler(domainHandler(addRR)))
	router.DELETE("/api/domains/:domain/rr", apiAuthHandler(domainHandler(delRR)))
}

func getDomains(_ *config.Options, u *happydns.User, p httprouter.Params, body io.Reader) Response {
	if domains, err := storage.MainStore.GetDomains(u); err != nil {
		return APIErrorResponse{
			err: err,
		}
	} else {
		return APIResponse{
			response: domains,
		}
	}
}

func addDomain(_ *config.Options, u *happydns.User, p httprouter.Params, body io.Reader) Response {
	var uz happydns.Domain
	err := json.NewDecoder(body).Decode(&uz)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	if len(uz.DomainName) <= 2 {
		return APIErrorResponse{
			err: errors.New("The given domain is invalid."),
		}
	}

	uz.DomainName = dns.Fqdn(uz.DomainName)

	if _, ok := dns.IsDomainName(uz.DomainName); !ok {
		return APIErrorResponse{
			err: fmt.Errorf("%q is not a valid domain name.", uz.DomainName),
		}
	}

	source, err := storage.MainStore.GetSource(u, uz.IdSource)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	if storage.MainStore.DomainExists(uz.DomainName) {
		return APIErrorResponse{
			err: errors.New("This domain has already been imported."),
		}

	} else if err := source.DomainExists(uz.DomainName); err != nil {
		return APIErrorResponse{
			err: err,
		}
	} else if err := storage.MainStore.CreateDomain(u, &uz); err != nil {
		return APIErrorResponse{
			err: err,
		}
	} else {
		return APIResponse{
			response: uz,
		}
	}
}

func delDomain(_ *config.Options, domain *happydns.Domain, body io.Reader) Response {
	if err := storage.MainStore.DeleteDomain(domain); err != nil {
		return APIErrorResponse{
			err: err,
		}
	} else {
		return APIResponse{
			response: true,
		}
	}
}

func domainHandler(f func(*config.Options, *happydns.Domain, io.Reader) Response) func(*config.Options, *happydns.User, httprouter.Params, io.Reader) Response {
	return func(opts *config.Options, u *happydns.User, ps httprouter.Params, body io.Reader) Response {
		if domain, err := storage.MainStore.GetDomainByDN(u, ps.ByName("domain")); err != nil {
			return APIErrorResponse{
				status: http.StatusNotFound,
				err:    errors.New("Domain not found"),
			}
		} else {
			return f(opts, domain, body)
		}
	}
}

func getDomain(_ *config.Options, domain *happydns.Domain, body io.Reader) Response {
	return APIResponse{
		response: domain,
	}
}

func axfrDomain(opts *config.Options, domain *happydns.Domain, body io.Reader) Response {
	source, err := storage.MainStore.GetSource(&happydns.User{Id: domain.IdUser}, domain.IdSource)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	rrs, err := source.ImportZone(domain)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	var ret []map[string]interface{}
	for _, rr := range rrs {
		ret = append(ret, map[string]interface{}{
			"string": rr.String(),
			"fields": rr,
		})
	}

	return APIResponse{
		response: ret,
	}
}

type uploadedRR struct {
	RR string `json:"string"`
}

func addRR(opts *config.Options, domain *happydns.Domain, body io.Reader) Response {
	var urr uploadedRR
	err := json.NewDecoder(body).Decode(&urr)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	rr, err := dns.NewRR(fmt.Sprintf("$ORIGIN %s\n$TTL %d\n%s", domain.DomainName, 3600, urr.RR))
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	source, err := storage.MainStore.GetSource(&happydns.User{Id: domain.IdUser}, domain.IdSource)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	err = source.AddRR(domain, rr)
	if err != nil {
		return APIErrorResponse{
			status: http.StatusInternalServerError,
			err:    err,
		}
	}

	return APIResponse{
		response: map[string]interface{}{
			"string": rr.String(),
		},
	}
}

func delRR(opts *config.Options, domain *happydns.Domain, body io.Reader) Response {
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

	source, err := storage.MainStore.GetSource(&happydns.User{Id: domain.IdUser}, domain.IdSource)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	err = source.DeleteRR(domain, rr)
	if err != nil {
		return APIErrorResponse{
			status: http.StatusInternalServerError,
			err:    err,
		}
	}

	return APIResponse{
		response: true,
	}
}
