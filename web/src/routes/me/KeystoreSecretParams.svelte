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
    import ResourceInput from "$lib/components/inputs/basic.svelte";
    import {
        KeystoreChoice,
        provideKeystoreChoice,
    } from "$lib/components/inputs/keystoreChoice.svelte";
    import type { Field } from "$lib/model/custom_form.svelte";

    // The parameters of a keystore keeping a secret of its own, such as the
    // token of a Vault: its owner chooses the keystore that secret goes to,
    // as for the secrets of any object, but never the keystore itself.
    interface Props {
        fields: Array<Field>;
        values: Record<string, unknown>;
        prefix: string;
        // The keystore updated, empty for one created.
        objectId?: string;
        keystore?: string;
    }

    let {
        fields,
        values = $bindable(),
        prefix,
        objectId,
        keystore = $bindable(""),
    }: Props = $props();

    const choice = provideKeystoreChoice(new KeystoreChoice("keystore", undefined, false));

    $effect(() => {
        choice.objectId = objectId;
        choice.exclude = objectId;
    });

    $effect(() => {
        keystore = choice.value;
    });
</script>

{#each fields as field (field.id)}
    <ResourceInput edit index={prefix + field.id} specs={field} bind:value={values[field.id!]} />
{/each}
