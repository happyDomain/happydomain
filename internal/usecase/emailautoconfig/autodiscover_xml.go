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

package emailautoconfig

import (
	"encoding/xml"
	"strings"

	"git.happydns.org/happyDomain/services/abstract"
)

// msAutodiscover matches Microsoft's POX (Plain Old XML) Autodiscover
// response schema. See https://learn.microsoft.com/en-us/exchange/architecture/client-access/autodiscover
type msAutodiscover struct {
	XMLName  xml.Name   `xml:"http://schemas.microsoft.com/exchange/autodiscover/responseschema/2006 Autodiscover"`
	Response msResponse `xml:"http://schemas.microsoft.com/exchange/autodiscover/outlook/responseschema/2006a Response"`
}

type msResponse struct {
	User    *msUser    `xml:"User,omitempty"`
	Account *msAccount `xml:"Account,omitempty"`
}

type msUser struct {
	DisplayName string `xml:"DisplayName,omitempty"`
}

type msAccount struct {
	AccountType string       `xml:"AccountType"`
	Action      string       `xml:"Action"`
	Protocols   []msProtocol `xml:"Protocol"`
}

type msProtocol struct {
	Type         string `xml:"Type"`
	Server       string `xml:"Server"`
	Port         uint16 `xml:"Port"`
	LoginName    string `xml:"LoginName,omitempty"`
	DomainName   string `xml:"DomainName,omitempty"`
	SSL          string `xml:"SSL"`
	Encryption   string `xml:"Encryption,omitempty"`
	SPA          string `xml:"SPA,omitempty"`
	AuthRequired string `xml:"AuthRequired"`
}

// msAutodiscoverIncomingType maps happyDomain incoming protocol identifiers
// to Microsoft Type vocabulary.
func msAutodiscoverIncomingType(protocol string) string {
	switch protocol {
	case "imap", "imaps":
		return "IMAP"
	case "pop3", "pop3s":
		return "POP3"
	}
	return strings.ToUpper(protocol)
}

// msAutodiscoverSSL returns "on" when the protocol uses TLS on connect,
// "off" otherwise.
func msAutodiscoverSSL(protocol string) string {
	switch protocol {
	case "imaps", "pop3s", "submissions":
		return "on"
	}
	return "off"
}

// msAutodiscoverEncryption returns the Encryption value for SMTP.
// "TLS" maps to STARTTLS in Outlook; "SSL" maps to TLS-on-connect.
func msAutodiscoverEncryption(protocol string) string {
	switch protocol {
	case "submissions":
		return "SSL"
	case "submission":
		return "TLS"
	}
	return "None"
}

func msLoginName(s *abstract.EmailAutoConfig) string {
	if s.UsernameFormat == "" {
		return "%EMAILADDRESS%"
	}
	return s.UsernameFormat
}

// RenderAutodiscoverXML returns a serialised Microsoft Autodiscover response
// for the given EmailAutoConfig service.
func RenderAutodiscoverXML(s *abstract.EmailAutoConfig, domainName, emailAddress string) ([]byte, error) {
	loginName := msLoginName(s)

	resp := msAutodiscover{
		Response: msResponse{
			Account: &msAccount{
				AccountType: "email",
				Action:      "settings",
			},
		},
	}

	if s.DisplayName != "" {
		resp.Response.User = &msUser{DisplayName: s.DisplayName}
	}

	if host := s.IncomingHost(); host != "" {
		proto := s.IncomingType()
		resp.Response.Account.Protocols = append(resp.Response.Account.Protocols, msProtocol{
			Type:         msAutodiscoverIncomingType(proto),
			Server:       host,
			Port:         s.IncomingPort(),
			LoginName:    loginName,
			DomainName:   domainName,
			SSL:          msAutodiscoverSSL(proto),
			SPA:          "off",
			AuthRequired: "on",
		})
	}

	if host := s.OutgoingHost(); host != "" {
		proto := s.OutgoingType()
		resp.Response.Account.Protocols = append(resp.Response.Account.Protocols, msProtocol{
			Type:         "SMTP",
			Server:       host,
			Port:         s.OutgoingPort(),
			LoginName:    loginName,
			DomainName:   domainName,
			SSL:          msAutodiscoverSSL(proto),
			Encryption:   msAutodiscoverEncryption(proto),
			SPA:          "off",
			AuthRequired: "on",
		})
	}

	if s.ExchangeServer != "" {
		resp.Response.Account.Protocols = append(resp.Response.Account.Protocols, msProtocol{
			Type:         "EXCH",
			Server:       s.ExchangeServer,
			SSL:          "on",
			AuthRequired: "on",
		})
	}

	body, err := xml.MarshalIndent(resp, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), body...), nil
}
