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

    import { Button, Icon, Input, InputGroup, Spinner } from "@sveltestrap/sveltestrap";

    import ResourceInput from "$lib/components/forms/ResourceInput.svelte";
    import type { Field } from "$lib/model/custom_form";
    import { t } from "$lib/translations";

    const dispatch = createEventDispatcher();

    export let edit = false;
    export let index: string;
    export let isNew = false;
    export let key: string;
    export let readonly = false;
    export let specs: Field;
    export let valuetype: string;
    export let value: any;

    let editKey = key == "";
    let initialKey = key;
    let renamingInProgress = false;
    let deletingInProgress = false;

    function rename() {
        if (key == "") {
            editKey = true;
            // TODO: throw error as key can't be empty
        } else if (key == initialKey) {
            editKey = false;
        } else {
            renamingInProgress = true;
            dispatch("rename-key", key);
        }
    }

    function deleteKey() {
        deletingInProgress = true;
        dispatch("delete-key", key);
    }
</script>

<h3>
    {#if editKey}
        <form on:submit|preventDefault={rename}>
            <InputGroup>
                <Input type="text" placeholder={specs.placeholder} bind:value={key} />
                <Button disabled={renamingInProgress} size="sm" color="primary">
                    {#if renamingInProgress}
                        <Spinner size="sm" />
                    {:else}
                        <Icon name="check" />
                    {/if}
                    {#if isNew}
                        {$t("domains.create-new-key", { id: specs.id })}
                    {:else}
                        {$t("common.rename")}
                    {/if}
                </Button>
            </InputGroup>
        </form>
    {:else}
        {key}
        {#if edit}
            <Button type="button" size="sm" color="link" on:click={() => (editKey = true)}>
                <Icon name="pencil" />
            </Button>
            <Button
                type="button"
                class="float-end"
                disabled={deletingInProgress}
                size="sm"
                color="danger"
                outline
                on:click={deleteKey}
            >
                {#if deletingInProgress}
                    <Spinner size="sm" />
                {:else}
                    <Icon name="trash" />
                {/if}
            </Button>
        {/if}
    {/if}
</h3>
<ResourceInput {edit} {index} {readonly} type={valuetype} bind:value />
