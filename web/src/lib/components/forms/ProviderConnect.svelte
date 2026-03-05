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
    import PForm from "$lib/components/forms/Provider.svelte";
    import ProviderSelector from "$lib/components/forms/ProviderSelector.svelte";
    import type { Provider } from "$lib/model/provider";
    import type { ProviderForm } from "$lib/model/provider_form.svelte.ts";
    import { t } from "$lib/translations";

    interface Props {
        /** Unique HTML form id — avoids conflicts when multiple modals coexist. */
        formId?: string;
        /** Bound by the parent so it can render SettingsStateButtons in its footer. */
        form?: ProviderForm;
        /** Bound by the parent so it can conditionalize its footer. */
        providerType?: string;
        /** Called when the provider has been successfully created. */
        ondone?: (provider: Provider) => void;
    }

    let {
        formId = "providerconnect",
        form = $bindable({} as ProviderForm),
        providerType = $bindable(""),
        ondone,
    }: Props = $props();

    function selectType(event: CustomEvent<{ ptype: string }>): void {
        providerType = event.detail.ptype;
    }

    function providerAdded(event: CustomEvent<Provider>): void {
        ondone?.(event.detail);
    }
</script>

{#if providerType}
    <PForm bind:form {formId} ptype={providerType} on:done={providerAdded} />
{:else}
    <ProviderSelector on:provider-selected={selectType} />
{/if}
