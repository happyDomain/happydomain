<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2026 happyDomain
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
    import BasicInput from "$lib/components/inputs/basic.svelte";
    import type { dnsResource } from "$lib/dns_rr";
    import { getRrtype, newRR } from "$lib/dns_rr";
    import { parseBIMI, stringifyBIMI } from "$lib/services/bimi";

    interface Props {
        dn: string;
        origin: Domain;
        value: dnsResource;
    }

    let { dn, origin, value = $bindable({}) }: Props = $props();

    if (!value["txt"]) {
        value["txt"] = newRR("default._bimi", getRrtype("TXT")) as any;
    }

    let val = $state(parseBIMI(value["txt"]!.Txt || ""));
    let selector = $state(
        value["txt"]!.Hdr?.Name?.replace("._bimi", "") || "default",
    );

    $effect(() => {
        const txt = value["txt"]!;
        txt.Txt = stringifyBIMI(val, txt.Txt || "");
        if (txt.Hdr) {
            txt.Hdr.Name = selector + "._bimi";
        }
    });

    const type = "svcs.BIMI";
</script>

<div>
    <h4 class="text-primary pb-1 border-bottom border-1">Brand Indicators for Message Identification</h4>

    <form id="addSvcForm">
        <BasicInput
            edit
            index="v"
            specs={{
                id: "v",
                label: "Version",
                placeholder: "BIMI1",
                type: "string",
                description: "Defines the version of BIMI to use.",
            }}
            bind:value={val.v}
        />

        <BasicInput
            edit
            index="selector"
            specs={{
                id: "selector",
                label: "Selector",
                placeholder: "default",
                type: "string",
                description: "Name of the BIMI record. Use 'default' unless you publish multiple logos.",
            }}
            bind:value={selector}
        />

        <BasicInput
            edit
            index="l"
            specs={{
                id: "l",
                label: "Logo",
                placeholder: "https://example.com/logo.svg",
                type: "string",
                description: "HTTPS URL of the SVG Tiny Portable/Secure logo.",
            }}
            bind:value={val.l}
        />

        <BasicInput
            edit
            index="a"
            specs={{
                id: "a",
                label: "Authority",
                placeholder: "https://example.com/vmc.pem",
                type: "string",
                description: "HTTPS URL of the Verified Mark Certificate (PEM). Required by Gmail and Yahoo.",
            }}
            bind:value={val.a}
        />

        <BasicInput
            edit
            index="e"
            specs={{
                id: "e",
                label: "Evidence",
                placeholder: "https://example.com/evidence",
                type: "string",
                description: "HTTPS URL of an evidence document (optional).",
            }}
            bind:value={val.e}
        />
    </form>
</div>
