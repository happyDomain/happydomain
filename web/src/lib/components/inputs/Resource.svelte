<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2024 happyDomain
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
    import { createEventDispatcher } from "svelte";

    import BasicInput from "$lib/components/inputs/basic.svelte";
    import MapInput from "$lib/components/inputs/map.svelte";
    import ObjectInput from "$lib/components/inputs/object.svelte";
    import RawInput from "$lib/components/inputs/raw.svelte";
    import TableInput from "$lib/components/inputs/table.svelte";
    import type { Field } from "$lib/model/custom_form.svelte";
    import type { ServiceInfos } from "$lib/model/service_specs.svelte";
    import type { CheckerCheckerOptionDocumentation } from "$lib/api-base/types.gen";

    const dispatch = createEventDispatcher();

    interface Props {
        edit?: boolean;
        editToolbar?: boolean;
        index?: string;
        noDecorate?: boolean;
        readonly?: boolean;
        showDescription?: boolean;
        specs?: Field | ServiceInfos | CheckerCheckerOptionDocumentation;
        type: string;
        value: any;
    }

    let {
        edit = false,
        editToolbar = false,
        index = "",
        noDecorate = false,
        readonly = false,
        showDescription = true,
        specs = undefined,
        type,
        value = $bindable(),
    }: Props = $props();

    function sanitizeType(t: string) {
        if (t.substring(0, 2) === "[]") t = t.substring(2);
        if (t.substring(0, 1) === "*") t = t.substring(1);
        return t;
    }
</script>

{#if specs && "hide" in specs && specs.hide}
    <!-- hidden input -->
{:else if type.substring(0, 2) === "[]" && type !== "[]byte" && type !== "[]uint8"}
    <TableInput
        edit={edit || editToolbar}
        {index}
        {noDecorate}
        {readonly}
        specs={specs as Field}
        type={sanitizeType(type)}
        bind:value
    />
{:else if type.substring(0, 3) === "map"}
    <MapInput
        edit={edit || editToolbar}
        {index}
        {readonly}
        specs={specs as Field}
        type={sanitizeType(type)}
        bind:value
    />
{:else if typeof value === "object" || Array.isArray(specs)}
    <ObjectInput
        {edit}
        {editToolbar}
        {index}
        {readonly}
        specs={specs as ServiceInfos}
        type={sanitizeType(type)}
        bind:value
        on:delete-this-service={(event) => dispatch("delete-this-service", event.detail)}
        on:update-this-service={(event) => dispatch("update-this-service", event.detail)}
    />
{:else if noDecorate}
    <RawInput
        edit={edit || editToolbar}
        {index}
        {readonly}
        specs={specs as Field}
        type={sanitizeType(type)}
        bind:value
    />
{:else}
    <BasicInput
        edit={edit || editToolbar}
        {index}
        {readonly}
        {showDescription}
        specs={specs as Field}
        type={sanitizeType(type)}
        bind:value
    />
{/if}
