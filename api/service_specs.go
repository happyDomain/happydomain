// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
// Authors: Pierre-Olivier Mercier, et al.
//
// This program is offered under a commercial and under the AGPL license.
// For commercial licensing, contact us at <contact@happydomain.org>.
//
// For AGPL licensing:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package api

import (
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/forms"
	"git.happydns.org/happyDomain/services"
)

func declareServiceSpecsRoutes(router *gin.RouterGroup) {
	router.GET("/service_specs", getServiceSpecs)

	router.GET("/service_specs/:ssid/icon.png", getServiceSpecIcon)

	apiServiceSpecsRoutes := router.Group("/service_specs/:ssid")
	apiServiceSpecsRoutes.Use(ServiceSpecsHandler)

	apiServiceSpecsRoutes.GET("", getServiceSpec)
}

// getServiceSpecs returns the static list of usable services in this happyDomain release.
//
//	@Summary	List all services with which you can connect.
//	@Schemes
//	@Description	This returns the static list of usable services in this happyDomain release.
//	@Tags			service_specs
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]svcs.ServiceInfos{}	"The list"
//	@Router			/service_specs [get]
func getServiceSpecs(c *gin.Context) {
	services := svcs.GetServices()

	ret := map[string]svcs.ServiceInfos{}
	for k, service := range *services {
		ret[k] = service.Infos
	}

	c.JSON(http.StatusOK, ret)
}

// getServiceSpecIcon returns the icon as image/png.
//
//	@Summary	Get the PNG icon.
//	@Schemes
//	@Description	Return the icon as a image/png file for the given service type.
//	@Tags			service_specs
//	@Accept			json
//	@Produce		png
//	@Param			serviceType	path		string	true	"The service's type"
//	@Success		200			{file}		png
//	@Failure		404			{object}	happydns.Error	"Service type does not exist"
//	@Router			/service_specs/{serviceType}/icon.png [get]
func getServiceSpecIcon(c *gin.Context) {
	ssid := string(c.Param("ssid"))

	if cnt, ok := svcs.Icons[strings.TrimSuffix(ssid, ".png")]; ok {
		c.Data(http.StatusOK, "image/png", cnt)
	} else {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Icon not found."})
	}
}

func ServiceSpecsHandler(c *gin.Context) {
	ssid := string(c.Param("ssid"))

	svc, err := svcs.FindSubService(ssid)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": fmt.Sprintf("Unable to find specs: %s", err.Error())})
		return
	}

	c.Set("servicetype", reflect.Indirect(reflect.ValueOf(svc)).Type())

	c.Next()
}

type viewServiceSpec struct {
	Fields []forms.Field `json:"fields,omitempty"`
}

func getSpecs(svcType reflect.Type) viewServiceSpec {
	fields := []forms.Field{}
	for i := 0; i < svcType.NumField(); i += 1 {
		if svcType.Field(i).Anonymous {
			fields = append(fields, getSpecs(svcType.Field(i).Type).Fields...)
			continue
		}

		jsonTag := svcType.Field(i).Tag.Get("json")
		jsonTuples := strings.Split(jsonTag, ",")

		f := forms.Field{
			Type: svcType.Field(i).Type.String(),
		}

		if len(jsonTuples) > 0 && len(jsonTuples[0]) > 0 {
			f.Id = jsonTuples[0]
		} else {
			f.Id = svcType.Field(i).Name
		}

		tag := svcType.Field(i).Tag.Get("happydomain")
		tuples := strings.Split(tag, ",")

		for _, t := range tuples {
			kv := strings.SplitN(t, "=", 2)
			if len(kv) > 1 {
				switch strings.ToLower(kv[0]) {
				case "label":
					f.Label = kv[1]
				case "placeholder":
					f.Placeholder = kv[1]
				case "default":
					var err error
					if strings.HasPrefix(f.Type, "uint") {
						f.Default, err = strconv.ParseUint(kv[1], 10, 64)
					} else if strings.HasPrefix(f.Type, "int") {
						f.Default, err = strconv.ParseInt(kv[1], 10, 64)
					} else if strings.HasPrefix(f.Type, "float") {
						f.Default, err = strconv.ParseFloat(kv[1], 64)
					} else if strings.HasPrefix(f.Type, "bool") {
						f.Default, err = strconv.ParseBool(kv[1])
					} else {
						f.Default = kv[1]
					}

					if err != nil {
						log.Printf("Format error for default field %s of type %s definition: %s", svcType.Field(i).Name, svcType.Name(), err.Error())
					}
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
				case "hidden":
					f.Hide = true
				default:
					f.Label = kv[0]
				}
			}
		}
		fields = append(fields, f)
	}

	return viewServiceSpec{fields}
}

// getServiceSpec returns a description of the expected fields.
//
//	@Summary	Get the service expected fields.
//	@Schemes
//	@Description	Return a description of the expected fields.
//	@Tags			service_specs
//	@Accept			json
//	@Produce		json
//	@Param			serviceType	path		string	true	"The service's type"
//	@Success		200			{object}	viewServiceSpec
//	@Failure		404			{object}	happydns.Error	"Service type does not exist"
//	@Router			/services/_specs/{serviceType} [get]
func getServiceSpec(c *gin.Context) {
	svctype := c.MustGet("servicetype").(reflect.Type)

	c.JSON(http.StatusOK, getSpecs(svctype))
}
