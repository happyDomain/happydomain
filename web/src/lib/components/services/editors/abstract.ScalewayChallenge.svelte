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
    import RecordLine from "$lib/components/services/editors/RecordLine.svelte";
    import type { Domain } from "$lib/model/domain";
    import type { dnsResource, dnsTypeTXT } from "$lib/dns_rr";
    import { getRrtype, newRR } from "$lib/dns_rr";

    interface Props {
        dn: string;
        origin: Domain;
        readonly?: boolean;
        value: dnsResource;
    }

    let { dn, origin, readonly = false, value = $bindable({}) }: Props = $props();
    const type = "abstract.ScalewayChallenge";

    // Initialize TXT record if needed
    if (!value["txt"]) {
        const txtRecord = newRR("_scaleway-challenge", getRrtype("TXT")) as dnsTypeTXT;
        txtRecord.Txt = "";
        value["txt"] = txtRecord;
    }

    // Scaleway challenge uses the TXT record content as-is
    let challengeToken = $state(value["txt"]?.Txt || "");

    // Sync challenge token back to TXT record
    $effect(() => {
        if (value["txt"]) {
            value["txt"].Hdr.Name = "_scaleway-challenge";
            value["txt"].Txt = challengeToken;
        }
    });
</script>

<div>
    <RecordLine {dn} {origin} bind:rr={value["txt"]!} />
    <BasicInput
        class="mt-3"
        edit
        index="challenge-token"
        specs={{
              id: "challenge-token",
              label: "Challenge Token",
              description: "The challenge token provided by Scaleway",
              type: "string",
              placeholder: "Enter the token from Scaleway",
          }}
        bind:value={challengeToken}
    />
</div>
