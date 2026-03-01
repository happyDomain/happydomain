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

    import { Badge, Button, ButtonGroup, Icon, Spinner, Table } from "@sveltestrap/sveltestrap";

    import { deleteDomain } from "$lib/api/domains";
    import ImgProvider from "$lib/components/providers/ImgProvider.svelte";
    import type { Domain } from "$lib/model/domain";
    import { navigate } from "$lib/stores/config";
    import { refreshDomains } from "$lib/stores/domains";
    import { providers_idx, providersSpecs, refreshProvidersSpecs } from "$lib/stores/providers";
    import { t } from "$lib/translations";
    import { getStatusColor, getStatusIcon } from "$lib/utils/check";

    interface Props {
        class?: ClassValue;
        items: Array<Domain>;
        [key: string]: any;
    }

    let { class: className, items, ...rest }: Props = $props();

    if (!$providersSpecs) refreshProvidersSpecs();

    async function delDomain(event: Event, item: Domain) {
        event.stopPropagation();

        if (!confirm($t("domains.alert.remove", { domain: item.domain }))) return;

        await deleteDomain(item.id);
        refreshDomains();
    }
</script>

{#if !items}
    <div class="d-flex gap-2 align-items-center justify-content-center my-3 {className}">
        <Spinner color="primary" />
        {$t("wait.retrieving-domains")}
    </div>
{:else if items.length === 0}
    <div class="text-center my-3 {className}">
        {$t("domains.filtered-no-result")}
    </div>
{:else}
    <Table class={className} striped hover responsive {...rest}>
        <thead>
            <tr>
                <th>{$t("common.domain")}</th>
                <th>{$t("domaingroups.title")}</th>
                <th>{$t("domains.view.provider")}</th>
                <th>Status</th>
                <th></th>
            </tr>
        </thead>
        <tbody>
            {#each items as item (item.id)}
                <tr
                    style="cursor: pointer"
                    onclick={() => navigate("/domains/" + encodeURIComponent(item.domain))}
                >
                    <td class="fw-semibold">{item.domain}</td>
                    <td>{item.group || ""}</td>
                    <td>
                        {#if $providers_idx && $providers_idx[item.id_provider]}
                            {@const provider = $providers_idx[item.id_provider]}
                            <a
                                href="/providers/{encodeURIComponent(item.id_provider)}"
                                class="d-flex align-items-center gap-2 text-decoration-none"
                            >
                                <ImgProvider
                                    id_provider={item.id_provider}
                                    style="max-width: 1.5em; max-height: 1.5em; object-fit: contain;"
                                />
                                {#if provider._comment}
                                    {provider._comment}
                                {:else if $providersSpecs && $providersSpecs[provider._srctype]}
                                    {$providersSpecs[provider._srctype].name}
                                {:else}
                                    {provider._srctype}
                                {/if}
                            </a>
                        {:else}
                            <em class="text-muted">{item.id_provider}</em>
                        {/if}
                    </td>
                    <td>
                        {#if item.last_check_status !== undefined}
                            <a
                                href="/domains/{encodeURIComponent(item.domain)}/checks"
                                class="text-decoration-none"
                            >
                                <Badge color={getStatusColor(item.last_check_status)}>
                                    <Icon name={getStatusIcon(item.last_check_status)} />
                                </Badge>
                            </a>
                        {/if}
                    </td>
                    <td class="text-end">
                        <ButtonGroup size="sm">
                            <Button
                                color="outline-secondary"
                                title={$t("domains.actions.view")}
                                onclick={(e) => {
                                    e.stopPropagation();
                                    navigate("/domains/" + encodeURIComponent(item.domain));
                                }}
                            >
                                <Icon name="eye" />
                            </Button>
                            <Button
                                color="outline-danger"
                                title={$t("domains.stop")}
                                onclick={(e) => delDomain(e, item)}
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
