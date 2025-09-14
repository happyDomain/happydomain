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

    import { Badge, Button, Icon, Input } from "@sveltestrap/sveltestrap";
    import { parseCAAParameter, stringifyCAAParameter } from "$lib/services/caa.svelte";

    const dispatch = createEventDispatcher();

    interface Props {
        edit?: boolean;
        readonly?: boolean;
        value?: string;
    }

    let { edit = $bindable(false), readonly = false, value = $bindable("") }: Props = $props();

    let val = $state(parseCAAParameter(value));

    $effect(() => {
        val = parseCAAParameter(value);
    });
    $effect(() => {
        value = stringifyCAAParameter(val);
    });
</script>

<Badge color="info" class="me-1">
    {#if edit}
        <form
            class="d-flex align-items-center gap-1"
            onsubmit={(e) => { e.preventDefault(); edit = false; }}
        >
            <Input bsSize="sm" placeholder="key" bind:value={val.Tag} />
            =
            <Input bsSize="sm" placeholder="value" bind:value={val.Value} />
            <Button tabindex={0} type="submit" color="success" size="sm">
                <Icon name="check" />
            </Button>
        </form>
    {:else}
        <span role="button" tabindex="0" ondblclick={() => (edit = true)} aria-label="Double-click to edit">
            {val.Tag}={val.Value}
        </span>
        <span
            role="button"
            tabindex="0"
            onclick={() => dispatch("delete-parameter")}
            onkeypress={() => dispatch("delete-parameter")}
        >
            <Icon name="x-circle-fill" />
        </span>
    {/if}
</Badge>
