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
    import type { Domain } from "$lib/model/domain";
    import RecordLine from "$lib/components/services/editors/RecordLine.svelte";
    import BasicInput from "$lib/components/inputs/basic.svelte";
    import { servicesSpecs } from "$lib/stores/services";
    import { parseKeyValueTxt } from "$lib/dns";

    interface Props {
        dn: string;
        origin: Domain;
        value: any;
    }

    let { dn, origin, value = $bindable({}) }: Props = $props();

    function parseMTASTS(val) {
        return parseKeyValueTxt(val);
    }
    function stringifyMTASTS(val) {
        const sep = (value["txt"].Txt.indexOf("; ") >= 0 ? "; " : ";");

        return "v=" + (val["v"] ? val["v"] : "STSv1") + (val["id"] ? sep + "id=" + val["id"] : "");
    }
    let val = $state(parseMTASTS(value["txt"].Txt));

    $effect(() => {
        val = parseMTASTS(value["txt"].Txt);
    });
    $effect(() => {
        value["txt"].Txt = stringifyMTASTS(val);
    });

    const type = "svcs.MTA_STS";
</script>

{#if $servicesSpecs && $servicesSpecs[type]}
    <p class="text-muted">
        {$servicesSpecs[type].description}
    </p>
{/if}
<div>
    <h4 class="text-primary pb-1 border-bottom border-1">MTA Strict Transport Security</h4>
    <RecordLine class="mb-4" {dn} {origin} bind:rr={value["txt"]} />

    <BasicInput
        edit
        specs={{
              id: "v",
              label: "Version",
              placeholder: "STSv1",
              type: "string",
              description: "Defines the version of STS to use.",
              }}
        bind:value={val.v}
    />

    <BasicInput
        edit
        specs={{
              id: "id",
              label: "Policy Identifier",
              placeholder: "20160831085700Z",
              type: "string",
              description: "A short string used to track policy updates.",
              }}
        bind:value={val.id}
    />
</div>
