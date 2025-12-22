// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
// Authors: David Dernoncourt, et al.
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

package providers // import "git.happydns.org/happyDomain/providers"

import (
	_ "github.com/StackExchange/dnscontrol/v4/providers/route53"

	"git.happydns.org/happyDomain/internal/adapters"
	"git.happydns.org/happyDomain/model"
)

type Route53API struct {
	KeyId         string `json:"key_id,omitempty" happydomain:"label=AWS Access Key ID,placeholder=AKIAIOSFODNN7EXAMPLE,description=AWS IAM access key ID (leave empty to use environment/instance credentials)"`
	SecretKey     string `json:"secret_key,omitempty" happydomain:"label=AWS Secret Access Key,placeholder=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY,secret,description=AWS IAM secret access key"`
	Token         string `json:"token,omitempty" happydomain:"label=Session Token,placeholder=,secret,description=Optional AWS STS session token"`
	RoleArn       string `json:"role_arn,omitempty" happydomain:"label=Role ARN,placeholder=arn:aws:iam::123456789012:role/DnsControlRole,description=Optional IAM role ARN to assume"`
	ExternalId    string `json:"external_id,omitempty" happydomain:"label=External ID,placeholder=,description=Optional external ID for role assumption"`
	DelegationSet string `json:"delegation_set,omitempty" happydomain:"label=Delegation Set ID,placeholder=N1PA6795SAMPLE,description=Optional reusable delegation set ID"`
}

func (s *Route53API) DNSControlName() string {
	return "ROUTE53"
}

func (s *Route53API) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *Route53API) ToDNSControlConfig() (map[string]string, error) {
	config := map[string]string{}

	if s.KeyId != "" {
		config["KeyId"] = s.KeyId
	}
	if s.SecretKey != "" {
		config["SecretKey"] = s.SecretKey
	}
	if s.Token != "" {
		config["Token"] = s.Token
	}
	if s.RoleArn != "" {
		config["RoleArn"] = s.RoleArn
	}
	if s.ExternalId != "" {
		config["ExternalId"] = s.ExternalId
	}
	if s.DelegationSet != "" {
		config["DelegationSet"] = s.DelegationSet
	}

	return config, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &Route53API{}
	}, happydns.ProviderInfos{
		Name:        "AWS Route 53",
		Description: "Amazon's highly available and scalable DNS web service with global anycast network.",
	}, RegisterProvider)
}
