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
    import { Button, Icon, Table, Spinner } from "@sveltestrap/sveltestrap";

    import { getServiceSpec } from "$lib/api/service_specs";
    import ResourceInput from "$lib/components/forms/ResourceInput.svelte";
    import type { Field } from "$lib/model/custom_form";
    import { t } from "$lib/translations";

    interface Props {
        edit?: boolean;
        index: string;
        noDecorate?: boolean;
        readonly?: boolean;
        specs: any;
        type: string;
        value: any;
    }

    let {
        edit = false,
        index,
        noDecorate = false,
        readonly = false,
        specs,
        type,
        value = $bindable()
    }: Props = $props();

    let linespecs: Array<Field> | null | undefined = $state(undefined);
    $effect(() => {
        getServiceSpec(type).then(
            (ss) => {
                linespecs = ss.fields;
            },
            () => {
                linespecs = null;
            },
        );
    });

    function addLine() {
        if (!value) value = [];
        value.push(linespecs ? {} : "");
        value = value;
    }

    function deleteLine(idx: number) {
        value.splice(idx, 1);
        value = value;
    }
</script>

{#if linespecs === undefined}
    <div class="d-flex justify-content-center">
        <Spinner color="primary" />
    </div>
{:else}
    {#if !noDecorate && specs && specs.label}
        <h4 class="mt-1 text-primary pb-1 border-bottom border-1">
            {specs.label}
            {#if specs.description}
                <small class="text-muted">
                    {specs.description}
                </small>
            {/if}
        </h4>
    {/if}
    <Table hover striped>
        <thead>
            <tr>
                {#if linespecs}
                    {#each linespecs as spec}
                        <th
                            >{#if spec.label}{spec.label}{:else}{spec.id}{/if}</th
                        >
                    {/each}
                {:else if specs}
                    <th
                        >{#if specs.label}{specs.label}{:else}{specs.id}{/if}</th
                    >
                {/if}
            </tr>
        </thead>
        <tbody>
            {#if value && value.length}
                {#each value as v, idx}
                    <tr>
                        {#if linespecs && linespecs.length}
                            {#each linespecs as spec}
                                <td>
                                    <ResourceInput
                                        {edit}
                                        noDecorate
                                        index={index + "_" + idx + "_" + spec.id}
                                        {readonly}
                                        specs={spec}
                                        type={spec.type}
                                        bind:value={value[idx][spec.id]}
                                    />
                                </td>
                            {/each}
                        {:else}
                            <td>
                                <ResourceInput
                                    {edit}
                                    noDecorate
                                    index={index + "_" + idx}
                                    {readonly}
                                    {type}
                                    bind:value={value[idx]}
                                />
                            </td>
                        {/if}
                        {#if edit}
                            <td>
                                <Button
                                    type="button"
                                    color="danger"
                                    outline
                                    size="sm"
                                    on:click={() => deleteLine(idx)}
                                >
                                    <Icon name="trash" />
                                </Button>
                            </td>
                        {/if}
                    </tr>
                {/each}
            {:else}
                <tr>
                    <td
                        colspan={(linespecs ? linespecs.length : 1) + (edit ? 1 : 0)}
                        class="fst-italic text-center"
                    >
                        {$t("common.no-content")}
                    </td>
                </tr>
            {/if}
        </tbody>
        {#if edit}
            <tfoot>
                <tr>
                    <td colspan={linespecs ? linespecs.length : 1}>
                        <Button type="button" color="primary" outline size="sm" on:click={addLine}>
                            <Icon name="plus" />
                            {$t("common.new-row")}
                        </Button>
                    </td>
                </tr>
            </tfoot>
        {/if}
    </Table>
{/if}
