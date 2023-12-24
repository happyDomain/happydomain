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
 import { goto } from '$app/navigation';

 import {
     Spinner,
 } from 'sveltestrap';

 import NewServicePath from '$lib/components/NewServicePath.svelte';
 import ServiceModal from '$lib/components/domains/ServiceModal.svelte';
 import { domains_idx } from '$lib/stores/domains';
 import { thisZone } from '$lib/stores/thiszone';
 import { t } from '$lib/translations';

 export let data: {domain: DomainInList; history: string; streamed: Object;};

 let selectedDomain = data.domain.domain;
 let selectedHistory: string | undefined;
 $: selectedHistory = data.history;
 $: if (!data.history && $domains_idx[selectedDomain] && $domains_idx[selectedDomain].zone_history && $domains_idx[selectedDomain].zone_history.length > 0) {
     selectedHistory = $domains_idx[selectedDomain].zone_history[0] as string;
 }
 $: if (selectedHistory && data.history != selectedHistory) {
     goto('/domains/' + encodeURIComponent(selectedDomain) + '/' + encodeURIComponent(selectedHistory));
 }

</script>

{#if data.history == selectedHistory}
    <slot />
{:else}
    <div class="mt-5 text-center flex-fill">
        <Spinner label="Spinning" />
        <p>{$t('wait.loading')}</p>
    </div>
{/if}

{#await data.streamed.zone then zone}
    <NewServicePath
        origin={data.domain}
        zone={zone}
    />
    <ServiceModal
        origin={data.domain}
        zone={zone}
        on:update-zone-services={(event) => thisZone.set(event.detail)}
    />
{/await}
