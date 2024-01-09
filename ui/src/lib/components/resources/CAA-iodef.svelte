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
 import { createEventDispatcher } from 'svelte';

 import {
     Button,
     Icon,
     Input,
 } from '@sveltestrap/sveltestrap';

 const dispatch = createEventDispatcher();

 export let newone = false;
 export let readonly = false;
 export let value: any;

 let kind: string = "web";
 let url: string;

 $: if (value) switch (value.split(":")[0]) {
     case "mailto":
         kind = "mail";
         url = value.split(":")[1];
         break;
     default:
         kind = "web";
         url = value;
 }

 function updateValue(url: string) {
     if (kind == "mail") {
         value = "mailto:" + url;
     } else {
         value = url;
     }
 }

 $: updateValue(url);
</script>

<div class="d-flex gap-2 mb-2">
    <Input type="select" bind:value={kind}>
        <option value="mail">Mail</option>
        <option value="web">Webhook</option>
    </Input>

    <Input type={kind == "mail" ? "email" : "text"} bind:value={url} />

    {#if !newone}
        <Button
            type="button"
            color="danger"
            outline
            on:click={() => dispatch("delete-iodef")}
        >
            <Icon name="trash" />
        </Button>
    {:else}
        <Button
            color="success"
            outline
            type="button"
            disabled={!value}
            on:click={() => {dispatch("add-iodef", value); value = { }}}
        >
            <Icon name="plus" />
        </Button>
    {/if}
</div>
