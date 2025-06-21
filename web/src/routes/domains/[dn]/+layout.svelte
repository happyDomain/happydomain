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
    import { goto, invalidateAll } from "$app/navigation";
    import { page } from "$app/state";

    // @ts-ignore
    import { escape } from "html-escaper";
    import {
        Button,
        ButtonDropdown,
        ButtonGroup,
        Col,
        Container,
        DropdownItem,
        DropdownMenu,
        DropdownToggle,
        Icon,
        Input,
        Row,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { deleteDomain as APIDeleteDomain } from "$lib/api/domains";
    import ButtonZonePublish from "./ButtonZonePublish.svelte";
    import ModalDiffZone from "./ModalDiffZone.svelte";
    import ModalDomainDelete, { controls as ctrlDomainDelete } from "./ModalDomainDelete.svelte";
    import ModalUploadZone, { controls as ctrlUploadZone } from "./ModalUploadZone.svelte";
    import ModalViewZone, { controls as ctrlViewZone } from "./ModalViewZone.svelte";
    import NewSubdomainPath, { controls as ctrlNewSubdomain } from "./NewSubdomainPath.svelte";
    import SelectDomain from "$lib/components/domains/SelectDomain.svelte";
    import SubdomainListTiny from "./SubdomainListTiny.svelte";
    import { fqdn, isReverseZone } from "$lib/dns";
    import type { Domain } from "$lib/model/domain";
    import type { ZoneMeta } from "$lib/model/zone";
    import { domains, domains_idx, refreshDomains } from "$lib/stores/domains";
    import {
        retrieveZone as StoreRetrieveZone,
        sortedDomains,
        sortedDomainsWithIntermediate,
        thisZone,
    } from "$lib/stores/thiszone";
    import { t } from "$lib/translations";

    interface Props {
        data: { domain: Domain };
        children?: import('svelte').Snippet;
    }

    let { data, children }: Props = $props();

    function domainLink(dn: string) : string {
        return $domains_idx[$domains_idx[dn].domain]
             ? $domains_idx[dn].domain
             : dn;
    }

    let selectedDomain = $derived(data.domain.id);
    function domainChange(dn: string) {
        if (dn != data.domain.id) {
            goto(
                "/domains/" +
                    encodeURIComponent(domainLink(dn)) +
                    (page.data.isAuditPage ? "/logs" : page.data.isHistoryPage ? "/history" : ""),
            );
        }
        if (selectedDomain != dn) {
            selectedDomain = dn;
        }
    }

    let selectedHistory: string | undefined = $derived(page.data.history);

    let retrievalInProgress = $state(false);
    async function retrieveZone() {
        retrievalInProgress = true;
        retrieveZoneDone(await StoreRetrieveZone(data.domain));
    }

    function retrieveZoneDone(zm: ZoneMeta): void {
        retrievalInProgress = false;
        if (page.data.definedhistory) {
            refreshDomains().then(() => {
                goto(
                    "/domains/" +
                        encodeURIComponent(domainLink(selectedDomain)) +
                        "/" +
                        encodeURIComponent(zm.id),
                );
            });
        } else {
            invalidateAll();
        }
    }

    function viewZone(): void {
        if (!selectedHistory) {
            return;
        }

        ctrlViewZone.Open(data.domain, selectedHistory);
    }

    let deleteInProgress = false;
    function detachDomain(): void {
        deleteInProgress = true;
        APIDeleteDomain($domains_idx[selectedDomain].id).then(
            () => {
                refreshDomains().then(
                    () => {
                        deleteInProgress = false;
                        goto("/domains");
                    },
                    () => {
                        deleteInProgress = false;
                        goto("/domains");
                    },
                );
            },
            (err: any) => {
                deleteInProgress = false;
                throw err;
            },
        );
    }
    $effect(() => {
        domainChange(selectedDomain);
    });
    $effect(() => {
        domainChange(data.domain.id);
    });

</script>

<Container fluid class="d-flex flex-column flex-fill">
    <Row class="flex-fill">
        <Col
            sm={4}
            md={3}
            class="py-2 sticky-top d-flex flex-column justify-content-between"
            style="background-color: #edf5f2; overflow-y: auto; max-height: 100vh; z-index: 0"
        >
            {#if $domains_idx[selectedDomain]}
                <div class="d-flex">
                    <Button href="/domains/" class="fw-bolder" color="link">
                        <Icon name="chevron-up" />
                    </Button>
                    <SelectDomain bind:selectedDomain={selectedDomain} />
                </div>

                {#if page.data.isHistoryPage || page.data.isAuditPage}
                    <Button
                        class="mt-2"
                        outline
                        color="primary"
                        href={"/domains/" + encodeURIComponent(domainLink(selectedDomain))}
                    >
                        <Icon name="chevron-left" />
                        {$t('zones.return-to')}
                    </Button>
                {:else}
                    <div class="d-flex gap-2 pb-2 sticky-top" style="padding-top: 10px">
                        <Button
                            type="button"
                            color="secondary"
                            outline
                            size="sm"
                            class="flex-fill"
                            disabled={!$sortedDomains}
                            on:click={() => ctrlNewSubdomain.Open()}
                        >
                            <Icon name="server" />
                            {$t("domains.add-a-subdomain")}
                        </Button>
                        <ButtonDropdown>
                            <DropdownToggle color="secondary" outline size="sm">
                                {#if retrievalInProgress}
                                    <Spinner size="sm" />
                                {:else}
                                    <Icon name="wrench-adjustable-circle" aria-hidden="true" />
                                {/if}
                            </DropdownToggle>
                            <DropdownMenu>
                                <DropdownItem header class="font-monospace">
                                    {data.domain.domain}
                                </DropdownItem>
                                <DropdownItem href={`/domains/${domainLink(selectedDomain)}/history`}>
                                    {$t("domains.actions.history")}
                                </DropdownItem>
                                <DropdownItem href={`/domains/${domainLink(selectedDomain)}/logs`}>
                                    {$t("domains.actions.audit")}
                                </DropdownItem>
                                <DropdownItem divider />
                                <DropdownItem on:click={viewZone} disabled={!$sortedDomains}>
                                    {$t("domains.actions.view")}
                                </DropdownItem>
                                <DropdownItem on:click={retrieveZone}>
                                    {$t("domains.actions.reimport")}
                                </DropdownItem>
                                <DropdownItem on:click={() => ctrlUploadZone.Open()}>
                                    {$t("domains.actions.upload")}
                                </DropdownItem>
                                <DropdownItem divider />
                                <DropdownItem disabled title="Coming soon...">
                                    {$t("domains.actions.share")}
                                </DropdownItem>
                                <DropdownItem on:click={() => ctrlDomainDelete.Open()}>
                                    {$t("domains.stop")}
                                </DropdownItem>
                                <DropdownItem divider />
                                <DropdownItem
                                    href={"/providers/" +
                                        encodeURIComponent(
                                            $domains_idx[selectedDomain].id_provider,
                                        )}
                                >
                                    {$t("provider.update")}
                                </DropdownItem>
                            </DropdownMenu>
                        </ButtonDropdown>
                    </div>
                    <div style="min-height:0; overflow-y: auto;" class="placeholder-glow">
                        {#if $sortedDomains && $sortedDomainsWithIntermediate && $thisZone && $thisZone.id == selectedHistory}
                            {#if $sortedDomains.length == 0}
                                <div class="text-truncate font-monospace text-muted">
                                    {data.domain.domain}
                                </div>
                            {:else if isReverseZone(data.domain.domain)}
                                <SubdomainListTiny domains={$sortedDomains} origin={data.domain} />
                            {:else}
                                <SubdomainListTiny
                                    domains={$sortedDomainsWithIntermediate}
                                    origin={data.domain}
                                />
                            {/if}
                        {:else}
                            <span class="d-block text-truncate font-monospace text-muted">
                                {data.domain.domain}
                            </span>
                            <span class="d-block placeholder ms-3 mb-1">
                                {data.domain.domain}
                            </span>
                            <span class="d-block placeholder ms-3 mb-1">
                                {data.domain.domain}
                            </span>
                            <span class="d-block placeholder ms-4 mb-1">
                                {data.domain.domain}
                            </span>
                            <span class="d-block placeholder ms-4 mb-1">
                                {data.domain.domain}
                            </span>
                            <span class="d-block placeholder ms-3">
                                {data.domain.domain}
                            </span>
                        {/if}
                    </div>
                {/if}

                <div class="flex-fill"></div>

                {#if !(page.data.isZonePage && data.domain.zone_history && $domains_idx[selectedDomain] && data.domain.id === $domains_idx[selectedDomain].id && $sortedDomainsWithIntermediate && selectedHistory)}
                    <Button
                        color="danger"
                        class="mt-3"
                        outline
                        on:click={() => ctrlDomainDelete.Open()}
                    >
                        <Icon name="trash" />
                        {$t("domains.stop")}
                    </Button>
                {:else}
                    <ButtonZonePublish
                        domain={data.domain}
                        history={selectedHistory}
                    />
                {/if}
            {:else}
                <div class="mt-4 text-center">
                    <Spinner color="primary" />
                </div>
            {/if}
        </Col>
        <Col sm={8} md={9} class="d-flex">
            {@render children?.()}
        </Col>
    </Row>
</Container>

<NewSubdomainPath origin={data.domain} />

<ModalUploadZone
    domain={data.domain}
    {selectedHistory}
    on:retrieveZoneDone={(ev) => retrieveZoneDone(ev.detail)}
/>

<ModalDomainDelete on:detachDomain={detachDomain} />

<ModalViewZone />

<ModalDiffZone
    domain={data.domain}
    {selectedHistory}
    on:retrieveZoneDone={(ev) => retrieveZoneDone(ev.detail)}
/>
