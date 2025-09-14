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
    import RecordText from "$lib/components/records/RecordText.svelte";
    import type { dnsRR } from "$lib/dns_rr";
    import type { Domain } from "$lib/model/domain";

    interface Props {
        class?: string;
        dn: string;
        origin: Domain;
        rrs: Array<dnsRR>;
    }

    let { class: className = "", dn, origin, rrs }: Props = $props();
</script>

<div class={className}>
    {#each rrs as rr}
        <RecordText {dn} {origin} {rr} />
    {/each}
</div>
