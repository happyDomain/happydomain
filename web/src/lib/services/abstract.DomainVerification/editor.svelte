<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2020-2026 happyDomain
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
    import type { Domain } from "$lib/model/domain";
    import type { AbstractDomainVerificationBody } from "$lib/services_bodies";
    import type { dnsTypeTXT } from "$lib/dns_rr";
    import { getRrtype, newRR } from "$lib/dns_rr";
    import {
        findVerificationProvider,
        selectVerificationProvider,
        setVerificationValue,
    } from "./model";
    import { verificationProviders } from "./providers";

    interface Props {
        dn: string;
        origin: Domain;
        readonly?: boolean;
        value: AbstractDomainVerificationBody;
    }

    let { dn, origin, readonly = false, value = $bindable() }: Props = $props();

    if (!value["txt"]) {
        value["txt"] = newRR("", getRrtype("TXT")) as dnsTypeTXT;
    }

    // A provider accepting several forms has several entries: it is offered
    // once, picking it applies its first form.
    const providerNames = [...new Set(verificationProviders.map((p) => p.name))];

    // The provider is not stored: it is told by the record.
    let provider = $derived(findVerificationProvider(value["txt"]));

    let where = $derived(
        provider?.label
            ? `The record is published on ${provider.label}${dn ? "." + dn : ""}.`
            : "The record is published on the domain itself.",
    );
</script>

<div>
    <BasicInput
        class="mt-3"
        edit
        index="provider"
        specs={{
            id: "provider",
            label: "Provider",
            description: provider ? where : "Pick the provider, or paste its value below.",
            type: "string",
            choices: providerNames,
        }}
        {readonly}
        bind:value={
            () => provider?.name ?? "",
            (name: string) => selectVerificationProvider(value["txt"], name)
        }
    />
    <BasicInput
        class="mt-3"
        edit
        index="verification-value"
        specs={{
            id: "verification-value",
            label: "Value",
            description: "The whole value the provider asks you to publish.",
            type: "string",
            placeholder: provider?.prefix ? provider.prefix + "…" : "google-site-verification=…",
        }}
        {readonly}
        bind:value={
            () => value["txt"].Txt,
            (txt: string) => setVerificationValue(value["txt"], txt ?? "")
        }
    />
</div>
