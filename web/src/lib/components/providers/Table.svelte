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
    import type { ClassValue } from "svelte/elements";

    import { Button, ButtonGroup, Icon, Spinner, Table } from "@sveltestrap/sveltestrap";

    import { deleteDomain } from "$lib/api/domains";
    import { deleteProvider } from "$lib/api/provider";
    import ImgProvider from "$lib/components/providers/ImgProvider.svelte";
    import NewProviderModal, {
        controls as newProviderControls,
    } from "$lib/components/modals/NewProvider.svelte";
    import type { Domain } from "$lib/model/domain";
    import type { Provider } from "$lib/model/provider";
    import { navigate } from "$lib/stores/config";
    import { domains, refreshDomains } from "$lib/stores/domains";
    import {
        providers,
        providersSpecs,
        refreshProviders,
        refreshProvidersSpecs,
    } from "$lib/stores/providers";
    import { t } from "$lib/translations";

    interface Props {
        class?: ClassValue;
        items: Array<Provider>;
        [key: string]: unknown;
    }

    let { class: className = "", items, ...rest }: Props = $props();

    if (!$providersSpecs) refreshProvidersSpecs();

    function domainsInProvider(
        domains: Array<Domain> | undefined,
        providers: Array<Provider> | undefined,
    ): Record<string, number> {
        const tmp: Record<string, number> = {};

        if (domains && providers) {
            for (const p of providers) {
                tmp[p._id] = 0;
            }
            for (const domain of domains) {
                if (!tmp[domain.id_provider]) {
                    tmp[domain.id_provider] = 0;
                }
                tmp[domain.id_provider]++;
            }
        }

        return tmp;
    }

    let domain_in_providers: Record<string, number> = $derived(
        domainsInProvider($domains, $providers),
    );

    function updateProvider(event: Event, item: Provider) {
        navigate("/providers/" + encodeURIComponent(item._id));
    }

    async function delProvider(event: Event, item: Provider) {
        event.stopPropagation();

        if ($domains) {
            const related_domains = $domains.filter((dn) => dn.id_provider == item._id);
            if (related_domains.length > 0) {
                if (
                    !confirm(
                        `There are ${related_domains.length} domains related to this provider, are you sure you want to delete all those domains too?`,
                    )
                )
                    return;

                for (const domain of related_domains) {
                    await deleteDomain(domain.id);
                }

                refreshDomains();
            }
        }

        await deleteProvider(item._id);
        refreshProviders();
    }
</script>

<NewProviderModal />

{#if !items || $providersSpecs == null}
    <div class="d-flex gap-2 align-items-center justify-content-center my-3 {className}">
        <Spinner color="primary" />
        {$t("wait.retrieving-providers")}
    </div>
{:else if items.length === 0}
    <div class="text-center my-3 {className}">
        <form
            onsubmit={(e) => {
                e.preventDefault();
                newProviderControls.Open();
            }}
        >
            {@html $t("provider.empty", {
                action: `<button type="submit" class="btn btn-link p-0">${$t("provider.empty-action")}</button>`,
            })}
        </form>
    </div>
{:else}
    <Table class={className} striped hover responsive {...rest}>
        <thead>
            <tr>
                <th>{$t("provider.provider-name")}</th>
                <th>{$t("provider.provider-type")}</th>
                <th>{$t("provider.linked-domains")}</th>
                <th></th>
            </tr>
        </thead>
        <tbody>
            {#each items as item (item._id)}
                <tr
                    style="cursor: pointer"
                    onclick={() => navigate("/providers/" + encodeURIComponent(item._id))}
                >
                    <td>
                        <div class="d-flex align-items-center gap-2">
                            <ImgProvider
                                ptype={item._srctype}
                                style="max-width: 2em; max-height: 2em; object-fit: contain;"
                            />
                            {#if item._comment}
                                <span title={item._comment}>{item._comment}</span>
                            {:else}
                                <em>{$t("provider.no-name")}</em>
                            {/if}
                        </div>
                    </td>
                    <td>
                        {#if $providersSpecs && $providersSpecs[item._srctype]}
                            {$providersSpecs[item._srctype].name}
                        {:else}
                            {item._srctype}
                        {/if}
                    </td>
                    <td>
                        <a href="/domains?provider={encodeURIComponent(item._id)}">
                            {domain_in_providers[item._id] ?? 0}
                        </a>
                    </td>
                    <td class="text-end">
                        <ButtonGroup size="sm">
                            <Button
                                color="outline-secondary"
                                title={$t("provider.update")}
                                onclick={(e) => updateProvider(e, item)}
                            >
                                <Icon name="pencil" />
                            </Button>
                            <Button
                                color="outline-danger"
                                title={$t("provider.delete")}
                                onclick={(e) => delProvider(e, item)}
                            >
                                <Icon name="trash" />
                            </Button>
                        </ButtonGroup>
                    </td>
                </tr>
            {/each}
        </tbody>
    </Table>
{/if}
