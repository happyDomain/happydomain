package api

import (
	"io"
	"reflect"
	"strings"

	"github.com/julienschmidt/httprouter"

	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/sources"
)

func init() {
	router.GET("/api/source_specs", apiHandler(getSourceSpecs))
	router.GET("/api/source_specs/*ssid", apiHandler(getSourceSpec))
}

type field struct {
	Label       string   `json:"label,omitempty"`
	Placeholder string   `json:"placeholder,omitempty"`
	Default     string   `json:"default,omitempty"`
	Choices     []string `json:"choices,omitempty"`
	Required    bool     `json:"required,omitempty"`
	Secret      bool     `json:"secret,omitempty"`
	Description string   `json:"description,omitempty"`
}

func getSourceSpecs(_ *config.Options, p httprouter.Params, body io.Reader) Response {
	srcs := sources.GetSources()

	ret := map[string]sources.SourceInfos{}
	for k, src := range *srcs {
		ret[k] = src.Infos
	}

	return APIResponse{
		response: ret,
	}
}

func getSourceSpec(_ *config.Options, p httprouter.Params, body io.Reader) Response {
	ssid := string(p.ByName("ssid"))
	if len(ssid) > 1 {
		ssid = ssid[1:]
	}

	src, err := sources.FindSource(ssid)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	srcType := reflect.Indirect(reflect.ValueOf(src)).Type()

	fields := []field{}
	for i := 0; i < srcType.NumField(); i += 1 {
		tag := srcType.Field(i).Tag.Get("happydns")
		tuples := strings.Split(tag, ",")
		f := field{}
		for _, t := range tuples {
			kv := strings.SplitN(t, "=", 2)
			if len(kv) > 1 {
				switch strings.ToLower(kv[0]) {
				case "label":
					f.Label = kv[1]
				case "placeholder":
					f.Placeholder = kv[1]
				case "default":
					f.Default = kv[1]
				case "description":
					f.Description = kv[1]
				case "choices":
					f.Choices = strings.Split(kv[1], ";")
				}
			} else {
				switch strings.ToLower(kv[0]) {
				case "required":
					f.Required = true
				case "secret":
					f.Secret = true
				default:
					f.Label = kv[0]
				}
			}
		}
		fields = append(fields, f)
	}

	return APIResponse{
		response: fields,
	}
}
