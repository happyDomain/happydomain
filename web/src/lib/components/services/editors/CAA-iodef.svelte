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

    import { Button, Icon, Input } from "@sveltestrap/sveltestrap";

    const dispatch = createEventDispatcher();

    interface Props {
        flag?: number;
        newone?: boolean;
        readonly?: boolean;
        tag?: string;
        value?: string;
    }

    let { flag = $bindable(0), newone = false, readonly = false, tag = $bindable(""), value = $bindable("") }: Props = $props();

    function parseCAAiodef(val) {
        const fields = val.split(":");

        return {
            kind: fields[0].replace(/s$/, ""),
            url: fields[0] == "mailto" ? fields.slice(1).join(":") : fields.join(":"),
        };
    }
    function stringifyCAAiodef(val) {
        return val.kind == "mailto" ? (val.kind + ":" + val.url) : val.url;
    }
    let val = $state(parseCAAiodef(value));

    $effect(() => {
        val = parseCAAiodef(value);
    });
    $effect(() => {
        if (val.kind == "mailto" || val.url.indexOf(":") > 0)
            value = stringifyCAAiodef(val);
    });
</script>

<div class="d-flex gap-2 mb-2">
    <Input type="select" bind:value={val.kind}>
        <option value="mailto">Mail</option>
        <option value="http">Webhook</option>
    </Input>

    <Input type={val.kind == "mailto" ? "email" : "text"} {readonly} bind:value={val.url} />

    {#if !readonly}
        {#if !newone}
            <Button type="button" color="danger" outline on:click={() => dispatch("delete-iodef")}>
                <Icon name="trash" />
            </Button>
        {:else}
            <Button
                color="success"
                outline
                type="button"
                disabled={!value}
                on:click={() => {
                    dispatch("add-iodef", value);
                    value = "";
                }}
            >
                <Icon name="plus" />
            </Button>
        {/if}
    {/if}
</div>
