// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
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

package controller

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/dnschecker"
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	checkerUC "git.happydns.org/happyDomain/internal/usecase/checker"
	"git.happydns.org/happyDomain/model"
)

func init() {
	dnschecker.RegisterChecker(&happydns.CheckerDefinition{
		ID:   "ctrl_secret_checker",
		Name: "ctrl_secret_checker",
		Options: happydns.CheckerOptionsDocumentation{
			UserOpts: []happydns.CheckerOptionDocumentation{
				{Id: "token", Type: "string", Secret: true},
				{Id: "plain", Type: "string"},
			},
		},
	})
}

// newSecretOptionsController returns a controller storing the options of
// checkerID, with secrets sealed under the instance policy, the user it acts
// for, and call, running handler on a request of that user for checkerID.
func newSecretOptionsController(t *testing.T, checkerID string) (*CheckerController, *happydns.User, func(method string, handler func(*gin.Context), body string, params gin.Params) *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	secrets, err := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db})
	if err != nil {
		t.Fatal(err)
	}
	optionsUC := checkerUC.NewCheckerOptionsUsecase(db, db).WithSecrets(secrets)
	cc := NewCheckerController(nil, optionsUC, nil, nil, nil, nil, false)
	user := &happydns.User{Id: happydns.Identifier("0123456789abcdef")}

	call := func(method string, handler func(*gin.Context), body string, params gin.Params) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(method, "/checkers/"+checkerID+"/options", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = append(gin.Params{{Key: "checkerId", Value: checkerID}}, params...)
		c.Set("LoggedUser", user)
		handler(c)
		return w
	}
	return cc, user, call
}

// Secret checker options never go back to the client, whatever the route.
func TestCheckerOptionsResponsesRedactSecrets(t *testing.T) {
	cc, user, call := newSecretOptionsController(t, "ctrl_secret_checker")

	for name, w := range map[string]*httptest.ResponseRecorder{
		"PUT options":    call(http.MethodPut, cc.ChangeCheckerOptions, `{"token":"t0p-s3cr3t","plain":"visible"}`, nil),
		"POST options":   call(http.MethodPost, cc.AddCheckerOptions, `{"plain":"still visible"}`, nil),
		"GET options":    call(http.MethodGet, cc.GetCheckerOptions, "", nil),
		"GET option":     call(http.MethodGet, cc.GetCheckerOption, "", gin.Params{{Key: "optname", Value: "token"}}),
		"PUT option":     call(http.MethodPut, cc.SetCheckerOption, `"n3w-s3cr3t"`, gin.Params{{Key: "optname", Value: "token"}}),
		"GET after":      call(http.MethodGet, cc.GetCheckerOptions, "", nil),
		"GET option new": call(http.MethodGet, cc.GetCheckerOption, "", gin.Params{{Key: "optname", Value: "token"}}),
	} {
		body := w.Body.String()
		if w.Code != http.StatusOK {
			t.Errorf("%s = %d %s", name, w.Code, body)
			continue
		}
		if strings.Contains(body, "s3cr3t") || strings.Contains(body, "hds:1:") {
			t.Errorf("%s leaks the secret: %s", name, body)
		}
	}

	merged, _, err := cc.OptionsUC.BuildMergedCheckerOptionsWithAutoFill("ctrl_secret_checker", &user.Id, nil, nil, nil)
	if err != nil || merged["token"] != "n3w-s3cr3t" {
		t.Errorf("run options = %v, %v; want the last secret in clear", merged, err)
	}
}

// ctrlTokenRule checks the format of the secret option token, as a checker
// validating an API key would.
type ctrlTokenRule struct{}

func (ctrlTokenRule) Name() string        { return "token_format" }
func (ctrlTokenRule) Description() string { return "checks the token format" }
func (ctrlTokenRule) Evaluate(_ context.Context, _ happydns.ObservationGetter, _ happydns.CheckerOptions) []happydns.CheckState {
	return []happydns.CheckState{{Status: happydns.StatusOK}}
}
func (ctrlTokenRule) ValidateOptions(opts happydns.CheckerOptions) error {
	if v, ok := opts["token"]; ok {
		if s, _ := v.(string); !strings.HasPrefix(s, "tok-") {
			return errors.New("token must start with tok-")
		}
	}
	return nil
}

func init() {
	dnschecker.RegisterChecker(&happydns.CheckerDefinition{
		ID:   "ctrl_validated_secret_checker",
		Name: "ctrl_validated_secret_checker",
		Options: happydns.CheckerOptionsDocumentation{
			UserOpts: []happydns.CheckerOptionDocumentation{
				{Id: "token", Type: "string", Secret: true},
				{Id: "plain", Type: "string"},
			},
		},
		Rules: []happydns.CheckRule{ctrlTokenRule{}},
	})
}

// Once a secret is stored, every route validating options sees it in clear:
// saving another option, or echoing the placeholder back, still passes a
// checker validating the secret's format.
func TestCheckerOptionsValidationSeesStoredSecret(t *testing.T) {
	cc, user, call := newSecretOptionsController(t, "ctrl_validated_secret_checker")

	steps := []struct {
		name string
		w    func() *httptest.ResponseRecorder
	}{
		{"PUT options", func() *httptest.ResponseRecorder {
			return call(http.MethodPut, cc.ChangeCheckerOptions, `{"token":"tok-1","plain":"a"}`, nil)
		}},
		{"POST options", func() *httptest.ResponseRecorder {
			return call(http.MethodPost, cc.AddCheckerOptions, `{"plain":"b"}`, nil)
		}},
		{"PUT option plain", func() *httptest.ResponseRecorder {
			return call(http.MethodPut, cc.SetCheckerOption, `"c"`, gin.Params{{Key: "optname", Value: "plain"}})
		}},
		{"PUT options echo", func() *httptest.ResponseRecorder {
			return call(http.MethodPut, cc.ChangeCheckerOptions, `{"token":"`+happydns.RedactedSecret+`","plain":"d"}`, nil)
		}},
		{"PUT option token echo", func() *httptest.ResponseRecorder {
			return call(http.MethodPut, cc.SetCheckerOption, `"`+happydns.RedactedSecret+`"`, gin.Params{{Key: "optname", Value: "token"}})
		}},
	}
	for _, step := range steps {
		if w := step.w(); w.Code != http.StatusOK {
			t.Errorf("%s = %d %s", step.name, w.Code, w.Body.String())
		}
	}

	merged, _, err := cc.OptionsUC.BuildMergedCheckerOptionsWithAutoFill("ctrl_validated_secret_checker", &user.Id, nil, nil, nil)
	if err != nil || merged["token"] != "tok-1" || merged["plain"] != "d" {
		t.Errorf("run options = %v, %v; want the stored token kept", merged, err)
	}
}
