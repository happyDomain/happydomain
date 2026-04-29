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

	"git.happydns.org/happyDomain/services/abstract"
)

// mozillaClientConfig matches the Thunderbird auto-configuration format
// (config-v1.1.xml). See https://wiki.mozilla.org/Thunderbird:Autoconfiguration:ConfigFileFormat
type mozillaClientConfig struct {
	XMLName       xml.Name             `xml:"clientConfig"`
	Version       string               `xml:"version,attr"`
	EmailProvider mozillaEmailProvider `xml:"emailProvider"`
}

type mozillaEmailProvider struct {
	ID               string                 `xml:"id,attr"`
	Domain           string                 `xml:"domain"`
	DisplayName      string                 `xml:"displayName,omitempty"`
	DisplayShortName string                 `xml:"displayShortName,omitempty"`
	IncomingServer   *mozillaIncomingServer `xml:"incomingServer,omitempty"`
	OutgoingServer   *mozillaOutgoingServer `xml:"outgoingServer,omitempty"`
}

type mozillaIncomingServer struct {
	Type           string `xml:"type,attr"`
	Hostname       string `xml:"hostname"`
	Port           uint16 `xml:"port"`
	SocketType     string `xml:"socketType"`
	Username       string `xml:"username"`
	Authentication string `xml:"authentication,omitempty"`
}

type mozillaOutgoingServer struct {
	Type           string `xml:"type,attr"`
	Hostname       string `xml:"hostname"`
	Port           uint16 `xml:"port"`
	SocketType     string `xml:"socketType"`
	Username       string `xml:"username"`
	Authentication string `xml:"authentication,omitempty"`
}

// mozillaSocketType returns the socketType value Thunderbird expects for a
// happyDomain protocol identifier. Plain protocols become "plain", TLS-on-
// connect become "SSL", and bare submission becomes "STARTTLS" (RFC 8314
// effectively deprecates plain submission).
func mozillaSocketType(protocol string) string {
	switch protocol {
	case "imaps", "pop3s", "submissions":
		return "SSL"
	case "submission":
		return "STARTTLS"
	default:
		return "plain"
	}
}

// mozillaIncomingType maps happyDomain protocol identifiers to the type
// attribute Thunderbird expects on <incomingServer>.
func mozillaIncomingType(protocol string) string {
	switch protocol {
	case "imap", "imaps":
		return "imap"
	case "pop3", "pop3s":
		return "pop3"
	}
	return protocol
}

// mozillaAuthentication maps happyDomain auth identifiers to Mozilla
// vocabulary. Most overlap; the empty default becomes "password-cleartext".
func mozillaAuthentication(auth string) string {
	if auth == "" {
		return "password-cleartext"
	}
	return auth
}

func mozillaUsernameFormat(s *abstract.EmailAutoConfig) string {
	if s.UsernameFormat == "" {
		return "%EMAILADDRESS%"
	}
	return s.UsernameFormat
}

// RenderMozillaXML returns a serialised Thunderbird config file for the
// given EmailAutoConfig service.
func RenderMozillaXML(s *abstract.EmailAutoConfig, domainName, emailAddress string) ([]byte, error) {
	id := domainName
	if id == "" {
		id = emailAddress
	}

	cfg := mozillaClientConfig{
		Version: "1.1",
		EmailProvider: mozillaEmailProvider{
			ID:               id,
			Domain:           domainName,
			DisplayName:      s.DisplayName,
			DisplayShortName: s.DisplayShortName,
		},
	}

	if host := s.IncomingHost(); host != "" {
		proto := s.IncomingType()
		cfg.EmailProvider.IncomingServer = &mozillaIncomingServer{
			Type:           mozillaIncomingType(proto),
			Hostname:       host,
			Port:           s.IncomingPort(),
			SocketType:     mozillaSocketType(proto),
			Username:       mozillaUsernameFormat(s),
			Authentication: mozillaAuthentication(s.IncomingAuth),
		}
	}

	if host := s.OutgoingHost(); host != "" {
		proto := s.OutgoingType()
		cfg.EmailProvider.OutgoingServer = &mozillaOutgoingServer{
			Type:           "smtp",
			Hostname:       host,
			Port:           s.OutgoingPort(),
			SocketType:     mozillaSocketType(proto),
			Username:       mozillaUsernameFormat(s),
			Authentication: mozillaAuthentication(s.OutgoingAuth),
		}
	}

	body, err := xml.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), body...), nil
}
