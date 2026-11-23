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

package providers_test

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	providerReg "git.happydns.org/happyDomain/internal/providerregistry"
	_ "git.happydns.org/happyDomain/providers"
)

// knownNameservers are actual name servers, along with the providers they
// belong to.
var knownNameservers = []struct {
	nameserver string
	providers  []string
}{
	{"a1-66.akam.net.", []string{"AkamaiEdgeDnsAPI"}},
	{"ns7.alidns.com.", []string{"AliDNSAPI"}},
	{"vip3.alidns.com.", []string{"AliDNSAPI"}},
	{"dns1.hichina.com.", []string{"AliDNSAPI"}},
	{"a.ns14.net.", []string{"AutoDNSAPI"}},
	{"any1.ns14.net.", []string{"AutoDNSAPI"}},
	{"ns1-39.azure-dns.com.", []string{"AzureDnsAPI"}},
	{"ns4-39.azure-dns.info.", []string{"AzureDnsAPI"}},
	{"kiki.bunny.net.", []string{"BunnyDNSAPI"}},
	{"kim.ns.cloudflare.com.", []string{"CloudflareAPI"}},
	{"pns64.cloudns.uk.", []string{"ClouDNSAPI"}},
	{"ns2.cloudns.net.", []string{"ClouDNSAPI"}},
	{"ns1.rrpproxy.net.", []string{"CNRAPI"}},
	{"dns1.cscdns.net.", []string{"CscGlobalAPI"}},
	{"udns2.cscudns.org.", []string{"CscGlobalAPI"}},
	{"ns1.desec.io.", []string{"DeSECAPI"}},
	{"ns2.desec.org.", []string{"DeSECAPI"}},
	{"ns1.digitalocean.com.", []string{"DigitalOceanAPI"}},
	{"ns1.dnscale.eu.", []string{"DNScaleAPI"}},
	{"ns2.dnsimple-edge.net.", []string{"DNSimpleAPI"}},
	{"ns1.dnsimple.com.", []string{"DNSimpleAPI"}},
	{"ns15.dnsmadeeasy.com.", []string{"DNSMadeEasyAPI"}},
	{"ns1.hyp.net.", []string{"DomainnameshopAPI"}},
	{"ns1.dreamhost.com.", []string{"DreamhostAPI"}},
	{"ns6.dynu.com.", []string{"DynuAPI"}},
	{"ns1.exoscale.ch.", []string{"ExoscaleAPI"}},
	{"ns1.exoscale.io.", []string{"ExoscaleAPI"}},
	{"ns-193-a.gandi.net.", []string{"GandiAPI"}},
	{"ns-100-c.gandi.net.", []string{"GandiAPI"}},
	{"ns-cloud-a1.googledomains.com.", []string{"GCloudAPI"}},
	{"ns1.gcorelabs.net.", []string{"GcoreAPI"}},
	{"ns2.gcdn.services.", []string{"GcoreAPI"}},
	{"dnsl1.gidinet.com.", []string{"GidinetAPI"}},
	{"dns5.gidinet.com.", []string{"GidinetAPI"}},
	{"ns1.gigahost.no.", []string{"GigahostAPI"}},
	{"ns71.domaincontrol.com.", []string{"GoDaddyAPI"}},
	{"pdns01.domaincontrol.com.", []string{"GoDaddyAPI"}},
	{"ns5.he.net.", []string{"HEDNSAPI"}},
	{"helium.ns.hetzner.de.", []string{"HetznerAPI"}},
	{"hydrogen.ns.hetzner.com.", []string{"HetznerAPI"}},
	{"ns1.hosting.de.", []string{"HostingdeAPI"}},
	{"ns1.dns-parking.com.", []string{"HostingerAPI"}},
	{"ns1.huaweicloud-dns.com.", []string{"HuaweiCloudAPI"}},
	{"ns11.infomaniak.ch.", []string{"InfomaniakAPI"}},
	{"nsany1.infomaniak.com.", []string{"InfomaniakAPI"}},
	{"ns-canada.topdns.com.", []string{"InternetbsAPI"}},
	{"ns.inwx.de.", []string{"INWXAPI"}},
	{"ns3.inwx.eu.", []string{"INWXAPI"}},
	{"ns1045.ui-dns.biz.", []string{"IonosAPI"}},
	{"x.ns.joker.com.", []string{"JokerAPI"}},
	{"ns1.linode.com.", []string{"LinodeAPI"}},
	{"ns1.loopia.se.", []string{"LoopiaAPI"}},
	{"ns4.luadns.net.", []string{"LuaDnsAPI"}},
	{"ns1.mythic-beasts.com.", []string{"MythicBeastsAPI"}},
	{"dns1.registrar-servers.com.", []string{"NamecheapAPI"}},
	{"ns1cvw.name.com.", []string{"NamedotcomAPI"}},
	{"ns1.name.com.", []string{"NamedotcomAPI"}},
	{"root-dns.netcup.net.", []string{"NetcupAPI"}},
	{"dns1.p01.nsone.net.", []string{"NS1API", "NetlifyAPI"}},
	{"ns1.nexdns.io.", []string{"NexDNSAPI"}},
	{"ns1.systemdns.com.", []string{"OpensrsAPI"}},
	{"ns1.p68.dns.oraclecloud.net.", []string{"OracleAPI"}},
	{"dns103.ovh.net.", []string{"OVHAPI"}},
	{"ns103.ovh.net.", []string{"OVHAPI"}},
	{"ns.ovh.net.", []string{"OVHAPI"}},
	{"dns200.anycast.me.", []string{"OVHAPI"}},
	{"ns1.packetframe.com.", []string{"PacketframeAPI"}},
	{"curitiba.ns.porkbun.com.", []string{"PorkbunAPI"}},
	{"ns1.yoursrs.com.", []string{"RealtimeRegisterAPI"}},
	{"ns-1982.awsdns-55.co.uk.", []string{"Route53API"}},
	{"ns-199.awsdns-24.com.", []string{"Route53API"}},
	{"zs1.rz.rwth-aachen.de.", []string{"RwthAPI"}},
	{"ns1.gslb3.sakura.ne.jp.", []string{"SakuraCloudAPI"}},
	{"ns0.dom.scw.cloud.", []string{"ScalewayAPI"}},
	{"ns1.softlayer.com.", []string{"SoftLayerAPI"}},
	{"launch1.spaceship.net.", []string{"SpaceshipAPI"}},
	{"ns0.transip.net.", []string{"TransIpAPI"}},
	{"ns2.transip.eu.", []string{"TransIpAPI"}},
	{"ns1.vercel-dns.com.", []string{"VercelAPI"}},
	{"ns4.vercel-dns-3.com.", []string{"VercelAPI"}},
	{"ns1.vultr.com.", []string{"VultrAPI"}},
	{"ns1.websupport.sk.", []string{"WebsupportAPI"}},

	// Not hosted by a provider happyDomain connects to, despite the likeness:
	// STRATO shares the infrastructure of IONOS, DFN serves many universities.
	{"ns-strato.ui-dns.com.", nil},
	{"dns-1.dfn.de.", nil},
	{"ns1.example.com.", nil},
}

func TestKnownNameservers(t *testing.T) {
	covered := map[string]bool{}

	for _, tt := range knownNameservers {
		got := providerReg.MatchNameservers([]string{tt.nameserver})
		if !slices.Equal(got, tt.providers) {
			t.Errorf("%s: got %v, want %v", tt.nameserver, got, tt.providers)
		}
		for _, p := range tt.providers {
			covered[p] = true
		}
	}

	for name, creator := range providerReg.GetProviders() {
		if len(creator.Infos.Nameservers) > 0 && !covered[name] {
			t.Errorf("%s: no name server of the provider in knownNameservers", name)
		}
	}
}

// TestNameserverPatterns catches the dots left unescaped, and the upper case
// that nothing can match.
func TestNameserverPatterns(t *testing.T) {
	bareDot := regexp.MustCompile(`(^|[^\\])\.`)

	for name, creator := range providerReg.GetProviders() {
		for _, pattern := range creator.Infos.Nameservers {
			if bareDot.MatchString(pattern) {
				t.Errorf("%s: unescaped dot in %q", name, pattern)
			}
			if pattern != strings.ToLower(pattern) {
				t.Errorf("%s: upper case in %q", name, pattern)
			}
		}
	}
}
