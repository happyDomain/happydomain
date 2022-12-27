// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydomain.org
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
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happydomain/forms"
	"git.happydns.org/happydomain/services"
)

func declareServiceSpecsRoutes(router *gin.RouterGroup) {
	router.GET("/service_specs", getServiceSpecs)

	router.GET("/service_specs/:ssid/icon.png", getServiceSpecIcon)

	apiServiceSpecsRoutes := router.Group("/service_specs/:ssid")
	apiServiceSpecsRoutes.Use(ServiceSpecsHandler)

	apiServiceSpecsRoutes.GET("", getServiceSpec)
}

func getServiceSpecs(c *gin.Context) {
	services := svcs.GetServices()

	ret := map[string]svcs.ServiceInfos{}
	for k, service := range *services {
		ret[k] = service.Infos
	}

	c.JSON(http.StatusOK, ret)
}

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

	return viewServiceSpec{fields}
}

func getServiceSpec(c *gin.Context) {
	svctype := c.MustGet("servicetype").(reflect.Type)

	c.JSON(http.StatusOK, getSpecs(svctype))
}
