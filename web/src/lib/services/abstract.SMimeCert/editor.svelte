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
    import { Alert } from "@sveltestrap/sveltestrap";

    import BasicInput from "$lib/components/inputs/basic.svelte";
    import type { Domain } from "$lib/model/domain";
    import { domainJoin } from "$lib/dns";
    import type { dnsResource, dnsTypeSMIMEA } from "$lib/dns_rr";
    import { getRrtype, newRR } from "$lib/dns_rr";
    import { t } from "$lib/translations";
    import { createEmailIdentifierHasher } from "$lib/utils/email_identifier.svelte";

    interface Props {
        dn: string;
        origin: Domain;
        readonly?: boolean;
        value: dnsResource & { username?: string; smimea?: dnsTypeSMIMEA };
    }

    let { dn, origin, readonly = false, value = $bindable({}) }: Props = $props();
    const type = "abstract.SMimeCert";

    // Initialize SMIMEA record if needed
    if (!value.smimea) {
        value.smimea = newRR("", getRrtype("SMIMEA")) as dnsTypeSMIMEA;
    }

    // Initialize username if not set
    if (value.username === undefined) {
        value.username = "";
    }

    // One-time init: extract existing name hash from domain name if no username
    let initialNameHash = "";
    if (!value["username"] && value["smimea"]?.Hdr?.Name) {
        const parts = value["smimea"].Hdr.Name.split("._smimecert");
        if (parts.length > 0 && parts[0]) {
            initialNameHash = parts[0];
        }
    }
    const hasher = createEmailIdentifierHasher(
        () => value["username"],
        () => origin.id,
        initialNameHash,
        value["username"] ?? "",
    );

    // When the name hash changes, update the domain name. A hash that had to be
    // dropped takes the owner name with it: keeping the old one would publish a
    // prefix that contradicts the username, which the backend rejects.
    $effect(() => {
        const hdr = value["smimea"]?.Hdr;
        if (!hdr) return;

        if (hasher.hash) {
            hdr.Name = domainJoin(hasher.hash, "_smimecert", dn);
        } else if (hasher.dropped) {
            hdr.Name = "";
        }
    });
</script>

<div>
    {#if hasher.error && !hasher.hash}
        <Alert color="warning" class="py-2 small mb-0 mt-3">
            {$t("errors.email-identifier", { error: hasher.error })}
        </Alert>
    {/if}

    <BasicInput
        class="mt-3"
        edit
        index="username"
        specs={{
            id: "username",
            label: "Username",
            description:
                "Email username (e.g., 'user' for user@domain.com). The SHA-224 hash will be computed automatically.",
            type: "string",
            placeholder: "user",
        }}
        bind:value={value["username"]}
    />

    <BasicInput
        edit={!value["username"] || hasher.error !== ""}
        index="name-hash"
        specs={{
            id: "name-hash",
            label: "Name Hash",
            description: value["username"]
                ? "SHA-224 hash computed from username (used as subdomain prefix)"
                : "SHA-224 hash of the username (28 bytes in hex). Edit directly or provide username above.",
            type: "string",
            placeholder: "c93f1e400f26708f98cb19d936620da35eec8f72e57f9eec01c1afd6",
        }}
        bind:value={hasher.hash}
    />

    <BasicInput
        edit
        index="certificate"
        specs={{
            id: "certificate",
            label: "Certificate",
            description: "Base64-encoded S/MIME certificate data",
            type: "string",
            placeholder: "Enter the S/MIME certificate",
        }}
        bind:value={value["smimea"]!.Certificate}
    />
</div>
