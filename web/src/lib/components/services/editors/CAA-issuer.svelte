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
    import { preventDefault } from 'svelte/legacy';

    import { Badge, Button, FormGroup, Icon, Input } from "@sveltestrap/sveltestrap";

    import { type dnsTypeCAA } from "$lib/dns_rr";
    import { t } from "$lib/translations";
    import CAAIssuerParameter from "./CAA-issuer-parameter.svelte";
    import { issuers, rev_issuers } from "./CAA-issuers";

    const dispatch = createEventDispatcher();

    interface Props {
        flag?: number;
        newone?: boolean;
        readonly?: boolean;
        tag?: string;
        value?: string;
    }

    let { flag = $bindable(0), newone = false, readonly = false, tag = $bindable(""), value = $bindable("") }: Props = $props();

    function parseCAA(val) {
        const fields = val.split(";");

        return {
            IssuerDomainName: !fields[0] && newone ? undefined : fields[0],
            Parameters: fields.length > 1 ? fields.slice(1) : [],
        };
    }
    function stringifyCAA(val) {
        const sep = (value && value.Value && value.Value.indexOf("; ") >= 0 ? "; " : ";");

        return val.IssuerDomainName == undefined ? "" : (val.IssuerDomainName + (val.Parameters.length ? sep + val.Parameters.join(sep) : ""));
    }
    let val = $state(parseCAA(value));

    $effect(() => {
        val = parseCAA(value);
    });
    $effect(() => {
        value = stringifyCAA(val);
    });

    const editable_parameters = $state({});
    function addParameter() {
        if (val.Parameters == null)
          val.Parameters = [];
        editable_parameters[val.Parameters.length] = true;
        val.Parameters.push("");
    }
</script>

<div class="d-flex gap-2 mb-2">
    {#if (newone && val.IssuerDomainName == undefined) || rev_issuers[val.IssuerDomainName]}
        <Input
            type="select"
            name="select"
            {readonly}
            bind:value={val.IssuerDomainName}
        >
            {#each Object.keys(issuers) as issuer}
                <option value={issuers[issuer][0]}>{issuer}</option>
            {/each}
            <option value={""}>{$t("common.other")}</option>
        </Input>
    {:else}
        <Input type="text" bind:value={val.IssuerDomainName} />
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
                value = "";
            }}
        >
            <Icon name="plus" />
        </Button>
    {/if}
</div>
{#if !newone}
    <div class="d-flex align-items-center">
        {#if val.Parameters}
            {#each val.Parameters as parameter, k}
                <CAAIssuerParameter
                    edit={editable_parameters[k]}
                    {readonly}
                    bind:value={val.Parameters[k]}
                    on:delete-parameter={() => val.Parameters.splice(k, 1)}
                />
            {/each}
        {/if}
        <span
            class="badge bg-primary"
            role="button"
            tabindex="0"
            onclick={addParameter}
            onkeypress={addParameter}
        >
            <Icon name="plus" /> Add parameter
        </span>
    </div>
{/if}
