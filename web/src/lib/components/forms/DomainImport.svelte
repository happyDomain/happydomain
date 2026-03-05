<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2026 happyDomain
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
    import type { Snippet } from "svelte";

    import CardImportableDomains from "$lib/components/providers/CardImportableDomains.svelte";
    import NewDomainInput from "$lib/components/inputs/NewDomain.svelte";
    import type { Provider } from "$lib/model/provider";

    interface Props {
        provider: Provider;
        /** Optional content rendered after NewDomainInput when the provider
         *  does not support domain listing (e.g. Onboarding's ZoneList). */
        extra?: Snippet;
    }

    let { provider, extra }: Props = $props();

    let noDomainsList = $state(false);
    let myDomain = $state("");
    let myDomainInProgress = $state(false);
</script>

<CardImportableDomains {provider} bind:noDomainsList />

{#if noDomainsList}
    <NewDomainInput
        bind:addingNewDomain={myDomainInProgress}
        autofocus
        class="mt-3"
        {provider}
        bind:value={myDomain}
    />
    {@render extra?.()}
{/if}
