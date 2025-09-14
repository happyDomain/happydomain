<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2025 happyDomain
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
    import {
        Button,
        Icon,
    } from "@sveltestrap/sveltestrap";
    import { emptyRR } from "$lib/dns";
    import type { dnsRR } from "$lib/dns_rr";
    import RecordText from "$lib/components/records/RecordText.svelte";
    import { controls } from "$lib/components/modals/Record.svelte";
    import type { Domain } from "$lib/model/domain";

    interface Props {
        class?: string;
        dn: string;
        origin: Domain;
        rr: dnsRR;
    }

    let { class: className = "", dn, origin, rr = $bindable(emptyRR()) }: Props = $props();

    function openEditor() {
        controls.Open(rr, dn);
    }
</script>

<div class="d-flex {className}">
    <RecordText
        class="flex-fill m-0 px-1 bg-light sticky-top pt-1 pb-1 border-1 border-bottom"
        {dn}
        {origin}
        bind:rr={rr}
    />
    <Button
        color="light"
        type="button"
        style="border-radius: 0 .25em .25em 0"
        on:click={openEditor}
    >
        <Icon name="pencil" />
    </Button>
</div>
