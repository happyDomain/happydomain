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
    import { navigate } from "$lib/stores/config";
    import { createEventDispatcher } from "svelte";

    import {
        Badge,
        Button,
        ButtonGroup,
        Dropdown,
        DropdownItem,
        DropdownMenu,
        DropdownToggle,
        Icon,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { deleteDomain } from "$lib/api/domains";
    import { deleteProvider } from "$lib/api/provider";
    import ImgProvider from "$lib/components/providers/ImgProvider.svelte";
    import HListGroup from "$lib/components/ListGroup.svelte";
    import type { Domain } from "$lib/model/domain";
    import type { Provider } from "$lib/model/provider";
    import { domains, refreshDomains } from "$lib/stores/domains";
    import {
        providers,
        providersSpecs,
        refreshProviders,
        refreshProvidersSpecs,
    } from "$lib/stores/providers";
    import { t } from "$lib/translations";

    const dispatch = createEventDispatcher();

    interface Props {
        flush?: boolean;
        noLabel?: boolean;
        noDropdown?: boolean;
        selectedProvider?: Provider | null;
        items: Array<any>;
        toolbar?: boolean;
        [key: string]: any
    }

    let {
        flush = false,
        noLabel = false,
        noDropdown = false,
        selectedProvider = $bindable(null),
        items,
        toolbar = false,
        ...rest
    }: Props = $props();

    if (!$providersSpecs) refreshProvidersSpecs();

    function domainsInProvider(domains: Array<Domain> | undefined, providers: Array<Provider> | undefined): Record<string, number> {
        const tmp: Record<string, number> = { };

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
    let domain_in_providers: Record<string, number> = $derived(domainsInProvider($domains, $providers));

    function selectProvider(event: CustomEvent<Provider>) {
        if (selectedProvider && selectedProvider._id == event.detail._id) {
            selectedProvider = null;
        } else {
            selectedProvider = event.detail;
            dispatch("click", selectedProvider);
        }
    }

    function updateProvider(event: Event, item: Provider) {
        event.stopPropagation();
        navigate("/providers/" + encodeURIComponent(item._id));
    }

    async function delProvider(event: Event, item: Provider) {
        event.stopPropagation();

        if ($domains) {
            // Check that there is no domain attached
            const related_domains = $domains.filter((dn) => dn.id_provider == item._id);
            if (related_domains.length > 0) {
                if (!confirm(`There are ${related_domains.length} domains related to this provider, are you sure you want to delete all those domains too?`)) return;

                for (const domain of related_domains) {
                    await deleteDomain(domain.id);
                }

                refreshDomains();
            }
        }

        await deleteProvider(item._id);
        refreshProviders();
    }

    function goNewProvider(e: Event) {
        e.preventDefault();
        dispatch("new-provider");
    }
</script>

{#if !items || $providersSpecs == null}
    <div class="d-flex gap-2 align-items-center justify-content-center my-3">
        <Spinner color="primary" />
        {$t("wait.retrieving-providers")}
    </div>
{:else}
    <HListGroup
        button
        {items}
        {flush}
        {...rest}
        isActive={(item) => selectedProvider != null && item._id == selectedProvider._id}
        on:click={selectProvider}
    >
        {#snippet empty()}
            <form onsubmit={goNewProvider}>
                {@html $t("provider.empty", {
                    action: `<button type="submit" class="btn btn-link p-0">${$t("provider.empty-action")}</button>`,
                  })}
            </form>
        {/snippet}
        {#snippet children({ item })}
            <div class="d-flex flex-fill justify-content-between" style="max-width: 100%">
                <div class="d-flex" style="min-width: 0">
                    <div class="text-center" style="width: 50px;">
                        <ImgProvider
                            ptype={item._srctype}
                            style="max-width: 100%; max-height: 2.5em; margin: -.6em .4em -.6em -.6em"
                        />
                    </div>
                    {#if item._comment}
                        <div class="text-truncate" title={item._comment}>
                            {item._comment}
                        </div>
                    {:else}
                        <em>{$t("provider.no-name")}</em>
                    {/if}
                </div>
                {#if !(noLabel && noDropdown && !toolbar)}
                    <div class="d-flex">
                        {#if !noLabel}
                            <div>
                                <Badge
                                    class="mx-1"
                                    color={domain_in_providers[item._id] > 0 ? "success" : "danger"}
                                >
                                    {$t("provider.associations", {
                                        count: domain_in_providers[item._id],
                                    })}
                                </Badge>
                                {#if $providersSpecs && $providersSpecs[item._srctype]}
                                    <Badge class="mx-1" color="secondary" title={item._srctype}>
                                        {$providersSpecs[item._srctype].name}
                                    </Badge>
                                {/if}
                            </div>
                        {/if}
                        {#if toolbar}
                            <ButtonGroup>
                                <Button
                                    color="light"
                                    size="sm"
                                    on:click={(e) => updateProvider(e, item)}
                                >
                                    <Icon name="pencil" />
                                </Button>
                                <Button color="light" size="sm" on:click={(e) => delProvider(e, item)}>
                                    <Icon name="trash" />
                                </Button>
                            </ButtonGroup>
                        {/if}
                        {#if !noDropdown}
                            <Dropdown size="sm" style="margin-right: -10px">
                                <DropdownToggle
                                    color="link"
                                    onclick={(event) => event.stopPropagation()}
                                >
                                    <Icon name="three-dots" />
                                </DropdownToggle>
                                <DropdownMenu>
                                    <DropdownItem on:click={(e) => updateProvider(e, item)}>
                                        {$t("provider.update")}
                                    </DropdownItem>
                                    <DropdownItem on:click={(e) => delProvider(e, item)}>
                                        {$t("provider.delete")}
                                    </DropdownItem>
                                </DropdownMenu>
                            </Dropdown>
                        {/if}
                    </div>
                {/if}
            </div>
        {/snippet}
    </HListGroup>
{/if}
