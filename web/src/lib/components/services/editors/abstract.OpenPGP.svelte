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
    import RecordLine from "$lib/components/services/editors/RecordLine.svelte";
    import BasicInput from "$lib/components/inputs/basic.svelte";
    import type { Domain } from "$lib/model/domain";
    import type { dnsResource, dnsTypeOPENPGPKEY } from "$lib/dns_rr";
    import { getRrtype, newRR } from "$lib/dns_rr";

    interface Props {
        dn: string;
        origin: Domain;
        readonly?: boolean;
        value: dnsResource & { username?: string; openpgpkey?: dnsTypeOPENPGPKEY };
    }

    let { dn, origin, readonly = false, value = $bindable({}) }: Props = $props();
    const type = "abstract.OpenPGP";

    // Initialize OPENPGPKEY record if needed
    if (!value.openpgpkey) {
        value.openpgpkey = newRR("", getRrtype("OPENPGPKEY")) as dnsTypeOPENPGPKEY;
    }

    // Initialize username if not set
    if (value.username === undefined) {
        value.username = "";
    }

    // Name hash state
    let nameHash = $state("");

    // Compute SHA-224 hash from username
    async function computeHash(username: string): Promise<string> {
        const encoder = new TextEncoder();
        const data = encoder.encode(username);
        const hashBuffer = await crypto.subtle.digest('SHA-256', data);
        const hashArray = new Uint8Array(hashBuffer);
        // Take first 28 bytes (224 bits) and convert to hex
        const hash224 = hashArray.slice(0, 28);
        return Array.from(hash224)
            .map(b => b.toString(16).padStart(2, '0'))
            .join('');
    }

    // When username changes, compute hash
    $effect(() => {
        if (value["username"]) {
            computeHash(value["username"]).then(hash => {
                nameHash = hash;
            });
        }
    });

    // Extract name hash from existing domain name on load
    $effect(() => {
        if (!value["username"] && value["openpgpkey"]?.Hdr?.Name && !nameHash) {
            const parts = value["openpgpkey"].Hdr.Name.split("._openpgpkey");
            if (parts.length > 0 && parts[0]) {
                nameHash = parts[0];
            }
        }
    });

    // When name hash changes, update the domain name
    $effect(() => {
        if (nameHash && value["openpgpkey"]?.Hdr) {
            value["openpgpkey"].Hdr.Name = nameHash + "._openpgpkey." + dn;
        }
    });
</script>

<div>
    {#if value["openpgpkey"]}
        <RecordLine {dn} {origin} bind:rr={value["openpgpkey"]} />
    {/if}

    <BasicInput
        class="mt-3"
        edit
        index="username"
        specs={{
            id: "username",
            label: "Username",
            description: "Email username (e.g., 'user' for user@domain.com). The SHA-224 hash will be computed automatically.",
            type: "string",
            placeholder: "user",
        }}
        bind:value={value["username"]}
    />

    <BasicInput
        edit={!value["username"]}
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
        bind:value={nameHash}
    />

    {#if value["openpgpkey"]}
        <BasicInput
            edit
            index="public-key"
            specs={{
                id: "public-key",
                label: "Public Key",
                description: "Base64-encoded OpenPGP public key data",
                type: "string",
                placeholder: "Enter the OpenPGP public key",
            }}
            bind:value={value["openpgpkey"].PublicKey}
        />
    {/if}
</div>
