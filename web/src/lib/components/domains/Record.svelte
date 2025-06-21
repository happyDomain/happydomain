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

    import { nsrrtype, rdatatostr } from '$lib/dns';
    import type { dnsRR } from '$lib/dns_rr';

    const dispatch = createEventDispatcher();

    interface Props {
        class: string;
        record: dnsRR;
    }

    let { class: className, record }: Props = $props();

    function openRecord() {
        dispatch('show-record', record);
    }
</script>

<div
    class="record d-flex gap-1 {className}"
    onclick={openRecord}
    onkeypress={openRecord}
>
    <span
        class="font-monospace text-truncate"
        title={rdatatostr(record)}
    >
        {record.Hdr.Name?record.Hdr.Name:'@'} {nsrrtype(record.Hdr.Rrtype)} {rdatatostr(record)}
    </span>
</div>

<style>
 .record:hover {
     background: #ccc;
 }
</style>
