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
    import { run } from 'svelte/legacy';

    import { Button, Icon } from "@sveltestrap/sveltestrap";

    import MapEntry from "./mapentry.svelte";
    import type { Field } from "$lib/model/custom_form";
    import { t } from "$lib/translations";

    const re = /^map\[(.*)\]\*?(.*)$/;

    interface Props {
        edit?: boolean;
        index: string;
        readonly?: boolean;
        specs: Field;
        type: string;
        value: any;
    }

    let {
        edit = false,
        index,
        readonly = false,
        specs,
        type,
        value = $bindable()
    }: Props = $props();

    let keytype: string | undefined = $state();
    let valuetype: string | undefined = $state();
    run(() => {
        const res = re.exec(type);
        if (res) {
            keytype = res[1];
            valuetype = res[2];
        }
    });
    run(() => {
        if (valuetype && !value) {
            value = {};
        }
    });

    function renameKey(oldkey: string, newkey: string) {
        value[newkey] = value[oldkey];
        delete value[oldkey];
        value = value;
    }

    function deleteKey(key: string) {
        delete value[key];
        value = value;
    }
</script>

{#if keytype && valuetype}
    {#if value && Object.keys(value).length}
        {#each Object.keys(value) as key}
            {#key key}
                <MapEntry
                    {edit}
                    {key}
                    index={index + "_" + key}
                    {readonly}
                    {specs}
                    {valuetype}
                    bind:value={value[key]}
                    on:delete-key={() => deleteKey(key)}
                    on:rename-key={(event) => renameKey(key, event.detail)}
                />
            {/key}
        {/each}
    {:else}
        <div class="my-2 text-center">
            {$t("common.no-thing", { thing: specs.label })}
        </div>
    {/if}
    {#if !("" in value)}
        <Button type="button" color="primary" on:click={() => (value[""] = {})}>
            <Icon name="plus" />
            {$t("common.add-new-thing", { thing: specs.label })}
        </Button>
    {/if}
{/if}
