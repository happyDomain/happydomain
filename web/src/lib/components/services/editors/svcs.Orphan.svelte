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
    import RecordLine from "$lib/components/services/editors/RecordLine.svelte";
    import RecordEditor from "$lib/components/records/Editor.svelte";
    import { servicesSpecs } from "$lib/stores/services";

    interface Props {
        dn: string;
        origin: Domain;
        readonly?: boolean;
        type?: string;
        value: any;
    }

    let {
        dn,
        origin,
        readonly = false,
        type = "svcs.Orphan",
        value = $bindable({}),
   }: Props = $props();
</script>

{#if $servicesSpecs && $servicesSpecs[type]}
    <p class="text-muted">
        {$servicesSpecs[type].description}
    </p>
{/if}
{#each Object.keys(value) as key}
    <RecordEditor
        bind:dn={dn}
        {origin}
        bind:record={value[key]}
    />
{/each}
