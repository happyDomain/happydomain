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
    import { invalidateAll } from "$app/navigation";
    import { navigate } from "$lib/stores/config";
    import { page } from "$app/state";

    import { Button, Col, Container, Icon, Row, Spinner } from "@sveltestrap/sveltestrap";

    import { deleteDomain as APIDeleteDomain } from "$lib/api/domains";
    import ChecksSidebarContent from "$lib/components/checkers/ChecksSidebarContent.svelte";
    import SelectDomain from "$lib/components/domains/SelectDomain.svelte";
    import type { Domain } from "$lib/model/domain";
    import type { ZoneMeta } from "$lib/model/zone";
    import { domainLink, domains_idx, refreshDomains } from "$lib/stores/domains";
    import { t } from "$lib/translations";
    import ButtonZonePublish from "./ButtonZonePublish.svelte";
    import ModalDiffZone from "./ModalDiffZone.svelte";
    import ModalDomainDelete, { controls as ctrlDomainDelete } from "./ModalDomainDelete.svelte";
    import ModalUploadZone from "./ModalUploadZone.svelte";
    import NewSubdomainPath from "./NewSubdomainPath.svelte";
    import ServiceDetailsOffcanvas from "./ServiceDetailsOffcanvas.svelte";
    import ServiceSidebar from "./ServiceSidebar.svelte";
    import ZoneSidebar from "./ZoneSidebar.svelte";
    import { thisZone } from "$lib/stores/thiszone";

    interface Props {
        data: { domain: Domain };
        children?: import("svelte").Snippet;
    }

    let { data, children }: Props = $props();

    let selectedDomain = $derived(data.domain.id);
    function domainChange(dn: string) {
        if (dn != data.domain.id) {
            navigate(
                "/domains/" +
                    encodeURIComponent(domainLink(dn)) +
                    (page.route.id
                        ? page.route.id.startsWith("/domains/[dn]/checks")
                            ? "/checks"
                            : page.route.id.startsWith("/domains/[dn]/logs")
                              ? "/logs"
                              : page.route.id.startsWith("/domains/[dn]/history")
                                ? "/history"
                                : page.route.id.startsWith("/domains/[dn]/[[historyid]]/export")
                                  ? "/export"
                                  : ""
                        : ""),
            );
        }
        if (selectedDomain != dn) {
            selectedDomain = dn;
        }
    }

    let selectedHistory: string | undefined = $derived(page.data.history);

    function retrieveZoneDone(zm: ZoneMeta): void {
        if (page.data.definedhistory) {
            refreshDomains().then(() => {
                navigate(
                    "/domains/" +
                        encodeURIComponent(domainLink(selectedDomain)) +
                        "/" +
                        encodeURIComponent(zm.id),
                );
            });
        } else {
            refreshDomains().then(() => {
                invalidateAll();
            });
        }
    }

    let deleteInProgress = $state(false);
    function detachDomain(): void {
        deleteInProgress = true;
        APIDeleteDomain($domains_idx[selectedDomain].id).then(
            () => {
                refreshDomains().then(
                    () => {
                        deleteInProgress = false;
                        navigate("/domains");
                    },
                    () => {
                        deleteInProgress = false;
                        navigate("/domains");
                    },
                );
            },
            (err: unknown) => {
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

<svelte:head>
    <title>{data.domain.domain} - happyDomain</title>
</svelte:head>

<Container fluid class="d-flex flex-column flex-fill">
    <Row class="flex-fill">
        <Col
            sm={4}
            md={3}
            class="py-2 sticky-top d-flex flex-column justify-content-between"
            style="background-color: #edf5f2; overflow-y: auto; max-height: 100vh; z-index: 0"
        >
            {#if $domains_idx[selectedDomain]}
                {@const isZonePage =
                    page.route.id &&
                    (page.route.id === "/domains/[dn]" ||
                        page.route.id === "/domains/[dn]/[[historyid]]")}
                <div class="d-flex">
                    <Button href={isZonePage ? "/domains/" : ".."} class="fw-bolder" color="link">
                        <Icon name={isZonePage ? "chevron-up" : "chevron-left"} />
                    </Button>
                    <SelectDomain bind:selectedDomain />
                </div>

                {#if page.route.id && page.route.id.startsWith("/domains/[dn]/checks")}
                    <ChecksSidebarContent
                        domain={data.domain}
                        checksBase={"/domains/" +
                            encodeURIComponent(domainLink(selectedDomain)) +
                            "/checks"}
                        backHref={"/domains/" + encodeURIComponent(domainLink(selectedDomain))}
                    />
                {:else if page.route.id && (page.route.id.startsWith("/domains/[dn]/history") || page.route.id.startsWith("/domains/[dn]/logs") || page.route.id.startsWith("/domains/[dn]/[[historyid]]/export"))}
                    <a
                        href="/domains/{encodeURIComponent(domainLink(selectedDomain))}"
                        class="sidebar-back d-flex align-items-center gap-1 mt-3 text-muted text-decoration-none fw-semibold"
                    >
                        <Icon name="chevron-left" />
                        {$t("zones.return-to")}
                    </a>
                {:else if page.route.id && page.route.id.startsWith("/domains/[dn]/[[historyid]]/[subdomain]/[serviceid]/checks")}
                    <ChecksSidebarContent
                        domain={data.domain}
                        checksBase={"/domains/" +
                            encodeURIComponent(domainLink(selectedDomain)) +
                            "/" +
                            encodeURIComponent(page.data.history ?? "") +
                            "/" +
                            encodeURIComponent(page.params.subdomain ?? "") +
                            "/" +
                            encodeURIComponent(page.data.serviceid ?? "") +
                            "/checks"}
                        backHref={"/domains/" +
                            encodeURIComponent(domainLink(selectedDomain)) +
                            "/" +
                            encodeURIComponent(page.data.history ?? "") +
                            "/" +
                            encodeURIComponent(page.params.subdomain ?? "") +
                            "/" +
                            encodeURIComponent(page.data.serviceid ?? "")}
                        serviceContext={{
                            zoneId: page.data.zoneId ?? "",
                            subdomain: page.data.subdomain ?? "",
                            serviceid: page.data.serviceid ?? "",
                        }}
                    />
                {:else if page.route.id === "/domains/[dn]/[[historyid]]/[subdomain]/[serviceid]"}
                    <ServiceSidebar
                        origin={data.domain}
                        subdomain={page.data.subdomain ?? ""}
                        serviceid={page.data.serviceid ?? ""}
                        historyId={page.data.history ?? ""}
                    />
                {:else}
                    <ZoneSidebar
                        origin={data.domain}
                        {selectedDomain}
                        {selectedHistory}
                        onretrieveZoneDone={retrieveZoneDone}
                    />
                {/if}

                {#if !(data.domain.zone_history && $domains_idx[selectedDomain] && data.domain.id === $domains_idx[selectedDomain].id && selectedHistory)}
                    <div class="flex-fill"></div>
                    <Button
                        color="danger"
                        class="mt-3"
                        outline
                        disabled={deleteInProgress}
                        on:click={() => ctrlDomainDelete.Open()}
                    >
                        {#if deleteInProgress}
                            <Spinner size="sm" />
                        {:else}
                            <Icon name="trash" />
                        {/if}
                        {$t("domains.stop")}
                    </Button>
                {:else if $domains_idx[data.domain.id] && $thisZone}
                    <div class="flex-fill"></div>
                    <ButtonZonePublish domain={data.domain} history={selectedHistory} />
                {/if}
            {:else}
                <div class="mt-4 text-center">
                    <Spinner color="primary" />
                </div>
            {/if}
        </Col>
        <div
            class="col-sm-8 col-md-9 d-flex"
            class:p-0={page.route &&
                (page.route.id == "/domains/[dn]/checks/[checkerId]/executions/[execId]" ||
                page.route.id == "/domains/[dn]/[[historyid]]/[subdomain]/[serviceid]/checks/[checkerId]/executions/[execId]")}
        >
            {@render children?.()}
        </div>
    </Row>
</Container>

<NewSubdomainPath origin={data.domain} />

<ModalUploadZone
    domain={data.domain}
    {selectedHistory}
    on:retrieveZoneDone={(ev) => retrieveZoneDone(ev.detail)}
/>

<ModalDomainDelete on:detachDomain={detachDomain} />

<ModalDiffZone domain={data.domain} {selectedHistory} />

<ServiceDetailsOffcanvas domain={data.domain} {selectedHistory} />
