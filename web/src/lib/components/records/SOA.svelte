<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2025 happyDomain
     Authors: Pierre-Olivier Mercier, et al.

     This program is offered under a commercial and under the AGPL license.
     For commercial licensing, contact us at <contact@happydomain.org>.

     For AGPL licensing:
     This program is free software: you can redistribute it and/or modify
     it under the terms of the GNU Affero General Public License as published by
     the Free Software Foundation, either version 3 of the License, or
     (at your option) any later version.

     This program is distributed in the hope that it will be useful,
     but WITHOUT ANY WARRANTY; without even the implied warranty of
     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
     GNU Affero General Public License for more details.

     You should have received a copy of the GNU Affero General Public License
     along with this program.  If not, see <https://www.gnu.org/licenses/>.
-->

<script lang="ts">
    import BasicInput from "$lib/components/inputs/basic.svelte";
    import type { dnsTypeSOA } from "$lib/dns_rr";

    interface Props {
        class: string;
        value?: dnsTypeSOA;
    }

    let { class: className, value = $bindable({} as dnsTypeSOA) }: Props = $props();
</script>

<div class={className}>
    <BasicInput
        edit
        index="soa_mname"
        specs={{
              id: "mname",
              label: "Name Server",
              placeholder: "ns0",
              type: "string",
              description: "The domain name of the name server that was the original or primary source of data for this zone.",
              }}
        bind:value={value.Ns}
    />

    <BasicInput
        edit
        index="soa_rname"
        specs={{
              id: "rname",
              label: "Contact Email",
              placeholder: "dnsmaster",
              type: "string",
              description: "A <domain-name> which specifies the mailbox of the person responsible for this zone.",
              }}
        bind:value={value.Mbox}
    />

    <BasicInput
        edit
        index="soa_serial"
        specs={{
              id: "serial",
              label: "Zone Serial",
              placeholder: "2147483647",
              type: "uint32",
              description: "The unsigned 32 bit version number of the original copy of the zone.  Zone transfers preserve this value.  This value wraps and should be compared using sequence space arithmetic.",
              }}
        bind:value={value.Serial}
    />

    <BasicInput
        edit
        index="soa_refresh"
        specs={{
              id: "refresh",
              label: "Slave Refresh Time",
              placeholder: "",
              type: "time.Duration",
              description: "The time interval before the zone should be refreshed by name servers other than the primary.",
              }}
        bind:value={value.Refresh}
    />

    <BasicInput
        edit
        index="soa_retry"
        specs={{
              id: "retry",
              label: "Retry Interval on failed refresh",
              placeholder: "",
              type: "time.Duration",
              description: "The time interval that should elapse before a failed refresh should be retried by a slave name server.",
              }}
        bind:value={value.Retry}
    />

    <BasicInput
        edit
        index="soa_expire"
        specs={{
              id: "expire",
              label: "Authoritative Expiry",
              placeholder: "",
              type: "time.Duration",
              description: "Time value that specifies the upper limit on the time interval that can elapse before the zone is no longer authoritative.",
              }}
        bind:value={value.Expire}
    />

    <BasicInput
        edit
        index="soa_nxttl"
        specs={{
              id: "nxttl",
              label: "Negative Caching Time",
              placeholder: "",
              type: "time.Duration",
              description: "Maximal time a resolver should cache a negative authoritative answer (such as NXDOMAIN ...).",
              }}
        bind:value={value.Minttl}
    />
</div>
