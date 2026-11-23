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
    import { Icon, ListGroup, ListGroupItem } from "@sveltestrap/sveltestrap";

    import ProviderConnect from "$lib/components/forms/ProviderConnect.svelte";
    import ImgProvider from "$lib/components/providers/ImgProvider.svelte";
    import type { Provider } from "$lib/model/provider";
    import type { ProviderForm } from "$lib/model/provider_form.svelte.ts";
    import { matchingProviders } from "$lib/provider_suggestion";
    import {
        providers,
        providersSpecs,
        refreshProviders,
        refreshProvidersSpecs,
    } from "$lib/stores/providers";
    import { t } from "$lib/translations";

    interface Props {
        /** Whether the "add new provider" sub-form is active. Bound by parent for footer logic. */
        addingProvider?: boolean;
        /** Bound by parent so it can render SettingsStateButtons in its footer. */
        form?: ProviderForm;
        /** Unique HTML form id — avoids conflicts when multiple pickers coexist. */
        formId?: string;
        /** Bound by parent so it can conditionalize its footer. */
        providerType?: string;
        /** Types of the providers to offer first; emptied when the user wants another one. */
        suggestedTypes?: string[];
        /** Called when a provider has been selected or created. */
        ondone?: (provider: Provider) => void;
    }

    let {
        addingProvider = $bindable(false),
        form = $bindable({} as ProviderForm),
        formId = "providerpicker",
        providerType = $bindable(""),
        suggestedTypes = $bindable([]),
        ondone,
    }: Props = $props();

    if (!$providersSpecs) refreshProvidersSpecs();

    function providerAdded(provider: Provider): void {
        refreshProviders();
        providerType = "";
        addingProvider = false;
        ondone?.(provider);
    }

    let offered = $derived(
        suggestedTypes.length
            ? matchingProviders($providers ?? [], suggestedTypes)
            : ($providers ?? []),
    );

    function typeName(ptype: string): string {
        return $providersSpecs?.[ptype]?.name ?? ptype;
    }

    function connect(ptype: string): void {
        // Left over when the same type has been given up before.
        form = {} as ProviderForm;
        providerType = ptype;
        addingProvider = true;
    }

    function useAnother(): void {
        suggestedTypes = [];
        // There is nothing to pick from: go straight to adding one.
        if (!$providers?.length) addingProvider = true;
    }
</script>

{#if addingProvider}
    <ProviderConnect bind:form bind:providerType {formId} ondone={providerAdded} />
{:else}
    <ListGroup>
        {#each offered as provider (provider._id)}
            <ListGroupItem
                tag="button"
                class="d-flex align-items-center gap-3"
                onclick={() => ondone?.(provider)}
            >
                <ImgProvider
                    id_provider={provider._id}
                    style="max-width: 2em; max-height: 2em; object-fit: contain; flex-shrink: 0;"
                />
                <div class="text-start">
                    <div class="fw-semibold">
                        {#if provider._comment}
                            {provider._comment}
                        {:else}
                            <em>{$t("provider.no-name")}</em>
                        {/if}
                    </div>
                    <small class="text-muted">
                        {typeName(provider._srctype)}
                    </small>
                </div>
                <Icon name="chevron-right" class="ms-auto text-muted" />
            </ListGroupItem>
        {/each}
        {#if suggestedTypes.length}
            {#each suggestedTypes as ptype (ptype)}
                <ListGroupItem
                    tag="button"
                    class="d-flex align-items-center gap-3 text-primary"
                    onclick={() => connect(ptype)}
                >
                    <Icon name="plus-circle" style="font-size: 1.5em; flex-shrink: 0;" />
                    <div class="text-start fw-semibold">
                        {$t("provider.suggestion.connect", { provider: typeName(ptype) })}
                    </div>
                </ListGroupItem>
            {/each}
            <ListGroupItem
                tag="button"
                class="d-flex align-items-center gap-3 text-muted"
                onclick={useAnother}
            >
                <Icon name="list-ul" style="font-size: 1.5em; flex-shrink: 0;" />
                <div class="text-start">
                    {$t("provider.another")}
                </div>
            </ListGroupItem>
        {:else}
            <ListGroupItem
                tag="button"
                class="d-flex align-items-center gap-3 text-primary"
                onclick={() => (addingProvider = true)}
            >
                <Icon name="plus-circle" style="font-size: 1.5em; flex-shrink: 0;" />
                <div class="text-start fw-semibold">
                    {$t("common.add-new-thing", { thing: $t("provider.kind") })}
                </div>
            </ListGroupItem>
        {/if}
    </ListGroup>
{/if}
