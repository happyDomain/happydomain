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

    import { Badge, Button, FormGroup, Icon, Input } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import { issuers, rev_issuers } from "./CAA-issuers";

    const dispatch = createEventDispatcher();

    export let newone = false;
    export let readonly = false;
    export let value: any = {};
</script>

<div class="d-flex gap-2 mb-2">
    {#if (newone && value.IssuerDomainName == undefined) || rev_issuers[value.IssuerDomainName]}
        <Input
            type="select"
            name="select"
            id="exampleSelect"
            {readonly}
            bind:value={value.IssuerDomainName}
        >
            {#each Object.keys(issuers) as issuer}
                <option value={issuers[issuer][0]}>{issuer}</option>
            {/each}
            <option value={""}>{$t("common.other")}</option>
        </Input>
    {:else}
        <Input type="text" bind:value={value.IssuerDomainName} />
    {/if}
    {#if !newone}
        <Button tabindex={0} type="button" color="danger" outline on:click={() => dispatch("delete-issuer")}>
            <Icon name="trash" />
        </Button>
    {:else}
        <Button
            color="success"
            tabindex={0}
            outline
            type="button"
            disabled={!value}
            on:click={() => {
                dispatch("add-issuer", value);
                value = {};
            }}
        >
            <Icon name="plus" />
        </Button>
    {/if}
</div>
{#if !newone}
    <div class="d-flex align-items-center">
        {#if value.Parameters}
            {#each value.Parameters as parameter, k}
                <Badge color="info" class="me-1">
                    {#if parameter.edit}
                        <form
                            class="d-flex align-items-center gap-1"
                            on:submit|preventDefault={() => (parameter.edit = false)}
                        >
                            <Input bsSize="sm" placeholder="key" bind:value={parameter.Tag} />
                            =
                            <Input bsSize="sm" placeholder="value" bind:value={parameter.Value} />
                            <Button tabindex={0} type="submit" color="success" size="sm">
                                <Icon name="check" />
                            </Button>
                        </form>
                    {:else}
                        <span role="button" tabindex="0" on:dblclick={() => (parameter.edit = true)} aria-label="Double-click to edit">
                            {parameter.Tag}={parameter.Value}
                        </span>
                        <span
                            role="button"
                            tabindex="0"
                            on:click={() => {
                                value.Parameters.splice(k, 1);
                                value = value;
                            }}
                            on:keypress={() => {
                                value.Parameters.splice(k, 1);
                                value = value;
                            }}
                        >
                            <Icon name="x-circle-fill" />
                        </span>
                    {/if}
                </Badge>
            {/each}
        {/if}
        <span
            class="badge bg-primary"
            role="button"
            tabindex="0"
            on:click={() => {
                if (value.Parameters == null) value.Parameters = [];
                value.Parameters.push({ Tag: "", Value: "", edit: true });
                value = value;
            }}
            on:keypress={() => {
                if (value.Parameters == null) value.Parameters = [];
                value.Parameters.push({ Tag: "", Value: "", edit: true });
                value = value;
            }}
        >
            <Icon name="plus" /> Add parameter
        </span>
    </div>
{/if}
