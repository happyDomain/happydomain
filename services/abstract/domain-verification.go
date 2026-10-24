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

package abstract

import (
	"errors"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	svc "git.happydns.org/happyDomain/internal/serviceanalyzer"
	"git.happydns.org/happyDomain/model"
)

// VerificationProvider describes the TXT record a third party asks for to
// prove that one controls a domain.
type VerificationProvider struct {
	Name string `json:"name"`

	// Label is the owner of the record, relative to the domain to verify;
	// empty when the record sits on the domain itself.
	Label string `json:"label,omitempty"`

	// Prefix is how the TXT value begins; empty when the provider takes any
	// value under its Label.
	Prefix string `json:"prefix,omitempty"`
}

// VerificationProviders lists the known providers. The first one matching a
// record wins. A provider accepting several forms has an entry for each, the
// first one being the form the editor proposes. Exported to the frontend by
// tools/gen_verification_providers.go.
var VerificationProviders = []VerificationProvider{
	{Name: "1Password", Prefix: "1password-site-verification="},
	{Name: "Adobe", Prefix: "adobe-idp-site-verification="},
	{Name: "Adobe Sign", Prefix: "adobe-sign-verification="},
	{Name: "Afternic", Prefix: "afternic-verification-"},
	{Name: "Ahrefs", Prefix: "ahrefs-site-verification_"},
	{Name: "Airtable", Prefix: "airtable-verification="},
	{Name: "Amazon SES", Label: "_amazonses"},
	{Name: "Amazon SES", Prefix: "amazonses:"},
	{Name: "Anthropic", Prefix: "anthropic-domain-verification-"},
	{Name: "Apple", Prefix: "apple-domain-verification="},
	{Name: "Atlassian", Prefix: "atlassian-domain-verification="},
	{Name: "Atlassian", Prefix: "atlassian-sending-domain-verification="},
	{Name: "Atlassian Statuspage", Prefix: "status-page-domain-verification="},
	{Name: "Autodesk", Prefix: "autodesk-domain-verification="},
	{Name: "Bluesky", Label: "_atproto", Prefix: "did="},
	{Name: "Box", Prefix: "box-domain-verification="},
	{Name: "Brevo", Prefix: "brevo-code:"},
	{Name: "Brevo", Prefix: "Sendinblue-code:"},
	{Name: "Brevo", Prefix: "sendinblue-site-verification="},
	{Name: "Calendly", Prefix: "calendly-site-verification="},
	{Name: "Canva", Prefix: "canva-site-verification="},
	{Name: "Cisco Webex", Prefix: "cisco-ci-domain-verification="},
	{Name: "Cisco Webex", Prefix: "webexdomainverification."},
	{Name: "Cursor", Prefix: "cursor-domain-verification-"},
	{Name: "dan.com", Prefix: "dan-ownership-verification="},
	{Name: "DigiCert", Label: "_dnsauth"},
	{Name: "Docker", Prefix: "docker-verification="},
	{Name: "DocuSign", Prefix: "docusign="},
	{Name: "Dropbox", Prefix: "dropbox-domain-verification="},
	{Name: "Facebook", Prefix: "facebook-domain-verification="},
	{Name: "Figma", Prefix: "figma-domain-verification="},
	{Name: "GitLab Pages", Label: "_gitlab-pages-verification-code", Prefix: "gitlab-pages-verification-code="},
	{Name: "GitLab Pages", Prefix: "gitlab-pages-verification-code="},
	{Name: "GlobalSign", Prefix: "_globalsign-domain-verification="},
	{Name: "GlobalSign", Prefix: "globalsign-domain-verification="},
	{Name: "GlobalSign", Prefix: "globalsign-smime-dv="},
	{Name: "Google", Prefix: "google-site-verification="},
	{Name: "HackerOne", Prefix: "h1-domain-verification="},
	{Name: "HashiCorp", Prefix: "hcp-domain-verification="},
	{Name: "Have I Been Pwned", Prefix: "have-i-been-pwned-verification="},
	{Name: "Have I Been Pwned", Prefix: "hibp-verify="},
	{Name: "HubSpot", Prefix: "hubspot-developer-verification="},
	{Name: "Jamf", Prefix: "jamf-site-verification="},
	{Name: "JetBrains", Prefix: "jetbrains-domain-verification="},
	{Name: "Keybase", Label: "_keybase", Prefix: "keybase-site-verification="},
	{Name: "Keybase", Prefix: "keybase-site-verification="},
	{Name: "Klaviyo", Prefix: "klaviyo-site-verification="},
	{Name: "KnowBe4", Prefix: "knowbe4-site-verification="},
	{Name: "Krisp", Prefix: "krisp-domain-verification="},
	{Name: "Linear", Prefix: "linear-domain-verification="},
	{Name: "LinkedIn", Prefix: "linkedin-site-verification="},
	{Name: "LogMeIn", Prefix: "logmein-verification-code="},
	{Name: "Loom", Prefix: "loom-site-verification="},
	{Name: "Loom", Prefix: "loom-verification="},
	{Name: "Mailgun", Prefix: "mgverify="},
	{Name: "Mandrill", Prefix: "mandrill_verify."},
	{Name: "Microsoft", Prefix: "MS="},
	{Name: "Microsoft", Prefix: "ms-domain-verification="},
	{Name: "Miro", Prefix: "miro-verification="},
	{Name: "Mixpanel", Prefix: "mixpanel-domain-verify="},
	{Name: "MongoDB", Prefix: "mongodb-site-verification="},
	{Name: "Notion", Prefix: "notion-domain-verification="},
	{Name: "Okta", Label: "_oktaverification"},
	{Name: "OneTrust", Prefix: "onetrust-domain-verification="},
	{Name: "OpenAI", Prefix: "openai-domain-verification="},
	{Name: "Pinterest", Prefix: "pinterest-site-verification="},
	{Name: "Postman", Prefix: "postman-domain-verification="},
	{Name: "Proton", Prefix: "protonmail-verification="},
	{Name: "Salesforce Marketing Cloud", Prefix: "SFMC-"},
	{Name: "Scaleway", Label: "_scaleway-challenge"},
	{Name: "Segment", Prefix: "segment-site-verification="},
	{Name: "Serval", Prefix: "serval-domain-verification-"},
	{Name: "Shopify", Prefix: "shopify-verification-code="},
	{Name: "Slack", Prefix: "slack-domain-verification="},
	{Name: "Smartsheet", Prefix: "smartsheet-site-validation="},
	{Name: "Stripe", Prefix: "stripe-verification="},
	{Name: "Tailscale", Prefix: "TAILSCALE-"},
	{Name: "TeamViewer", Prefix: "teamviewer-sso-verification="},
	{Name: "TikTok", Prefix: "tiktok-developers-site-verification="},
	{Name: "TikTok", Prefix: "tiktok-domain-verification="},
	{Name: "Tuta", Prefix: "t-verify="},
	{Name: "Twilio", Label: "_twilio", Prefix: "twilio-domain-verification="},
	{Name: "Twilio", Prefix: "twilio-domain-verification="},
	{Name: "Vercel", Label: "_vercel", Prefix: "vc-domain-verify="},
	{Name: "Vercel", Prefix: "vercel-domain-verification-"},
	{Name: "VMware", Prefix: "vmware-cloud-verification-"},
	{Name: "Workplace from Meta", Prefix: "workplace-domain-verification="},
	{Name: "Wrike", Prefix: "wrike-verification="},
	{Name: "Yahoo", Prefix: "yahoo-verification-key="},
	{Name: "Yandex", Prefix: "yandex-verification:"},
	{Name: "Zapier", Prefix: "zapier-domain-verification-challenge="},
	{Name: "Zoho", Prefix: "zoho-verification="},
	{Name: "Zoom", Prefix: "ZOOM_verify_"},
	{Name: "Zoom", Prefix: "zoom-domain-verification="},
}

// verificationProviderNames lists each provider once, in the order of the
// table, for the service to be found by any of them.
func verificationProviderNames() (names []string) {
	seen := map[string]bool{}
	for _, p := range VerificationProviders {
		if !seen[p.Name] {
			seen[p.Name] = true
			names = append(names, p.Name)
		}
	}
	return
}

// findVerificationProvider returns the provider of a TXT record whose owner
// is label, relative to the domain to verify.
func findVerificationProvider(label, txt string) *VerificationProvider {
	for i, p := range VerificationProviders {
		if p.Label == label && strings.HasPrefix(txt, p.Prefix) {
			return &VerificationProviders[i]
		}
	}
	return nil
}

// DomainVerification is the TXT record proving to a third party that one
// controls the domain. The provider is not stored: it is told by the record.
type DomainVerification struct {
	Record *happydns.TXT `json:"txt"`
}

func (s *DomainVerification) GetNbResources() int {
	return 1
}

func (s *DomainVerification) GenComment() string {
	if s.Record == nil {
		return ""
	}
	if p := findVerificationProvider(s.Record.Hdr.Name, s.Record.Txt); p != nil {
		return p.Name
	}
	return ""
}

func (s *DomainVerification) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	if s.Record == nil {
		return nil, errors.New("domain verification: no TXT record")
	}
	return []happydns.Record{s.Record}, nil
}

func domainverification_analyze(a *svc.Analyzer) error {
	for _, record := range a.SearchRR(svc.AnalyzerRecordFilter{Type: dns.TypeTXT}) {
		txt, ok := record.(*happydns.TXT)
		if !ok {
			continue
		}

		for _, p := range VerificationProviders {
			domain := record.Header().Name
			if p.Label != "" {
				var found bool
				if domain, found = strings.CutPrefix(domain, p.Label+"."); !found {
					continue
				}
			}

			if !strings.HasPrefix(txt.Txt, p.Prefix) {
				continue
			}

			if err := a.UseRR(record, domain, &DomainVerification{
				Record: helpers.RRRelativeSubdomain(record, a.GetOrigin(), domain).(*happydns.TXT),
			}); err != nil {
				return err
			}
			break
		}
	}
	return nil
}

func init() {
	svc.RegisterService(
		func() happydns.ServiceBody {
			return &DomainVerification{}
		},
		domainverification_analyze,
		happydns.ServiceInfos{
			Name:   "Domain Verification",
			Family: happydns.SERVICE_FAMILY_ABSTRACT,
			Categories: []string{
				"verification",
			},
			Keywords: verificationProviderNames(),
			Restrictions: happydns.ServiceRestrictions{
				NearAlone: true,
			},
		},
		2,
		"abstract.GoogleVerif",
		"abstract.KeybaseVerif",
		"abstract.GitlabPageVerif",
		"abstract.ScalewayChallenge",
	)
}
