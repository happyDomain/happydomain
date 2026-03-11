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
    // SvelteKit imports
    import { invalidateAll } from "$app/navigation";
    import { page } from "$app/state";

    // Component imports
    import { Button, Col, Container, Icon, Row, Spinner } from "@sveltestrap/sveltestrap";

    // Store imports
    import { navigate } from "$lib/stores/config";
    import { domains_idx, refreshDomains } from "$lib/stores/domains";

    // Utility imports
    import { t } from "$lib/translations";

    // Model imports
    import type { Domain } from "$lib/model/domain";
    import type { ZoneMeta } from "$lib/model/zone";

    // API imports
    import { deleteDomain as APIDeleteDomain } from "$lib/api/domains";

    // Local components
    import ChecksSidebarContent from "$lib/components/checkers/ChecksSidebarContent.svelte";
    import SelectDomain from "$lib/components/domains/SelectDomain.svelte";
    import ButtonZonePublish from "./ButtonZonePublish.svelte";
    import ModalDiffZone from "./ModalDiffZone.svelte";
    import ModalDomainDelete, { controls as ctrlDomainDelete } from "./ModalDomainDelete.svelte";
    import ModalUploadZone from "./ModalUploadZone.svelte";
    import NewSubdomainPath from "./NewSubdomainPath.svelte";
    import ServiceDetailsOffcanvas from "./ServiceDetailsOffcanvas.svelte";
    import ServiceSidebar from "./ServiceSidebar.svelte";
    import ZoneSidebar from "./ZoneSidebar.svelte";

    // Props
    interface Props {
        data: { domain: Domain };
        children?: import("svelte").Snippet;
    }

    let { data, children }: Props = $props();

    // Derived values
    let selectedHistory: string | undefined = $derived(page.data.history);

    // Local state
    let selectedDomain = $state(data.domain.id);
    let deleteInProgress = $state(false);

    // Functions
    function domainLink(dn: string): string {
        return $domains_idx[$domains_idx[dn].domain] ? $domains_idx[dn].domain : dn;
    }

    function domainChange(dn: string) {
        navigate(
            "/domains/" +
                encodeURIComponent(domainLink(dn)) +
                (page.route.id
                    ? page.route.id.startsWith("/domains/[dn]/logs")
                        ? "/logs"
                        : page.route.id.startsWith("/domains/[dn]/history")
                          ? "/history"
                          : page.route.id.startsWith("/domains/[dn]/[[historyid]]/export")
                            ? "/export"
                            : page.route.id.startsWith("/domains/[dn]/checks/[cname]")
                              ? `/checks/${page.params.cname!}`
                              : page.route.id.startsWith("/domains/[dn]/checks")
                                ? "/checks"
                                : ""
                    : ""),
        );
    }

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
            invalidateAll();
        }
    }

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
            (err: any) => {
                deleteInProgress = false;
                throw err;
            },
        );
    }

    // Effects
    $effect(() => {
        // Navigate when user selects a different domain from the dropdown
        if (selectedDomain !== data.domain.id) {
            domainChange(selectedDomain);
        }
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
                    <Button href={isZonePage ? "/domains/" : "."} class="fw-bolder" color="link">
                        <Icon name={isZonePage ? "chevron-up" : "chevron-left"} />
                    </Button>
                    <SelectDomain bind:selectedDomain />
                </div>

                {#if page.route.id && page.route.id.startsWith("/domains/[dn]/[[historyid]]/[subdomain]/[serviceid]/checks/")}
                    {@const serviceChecksBase = `/domains/${encodeURIComponent(domainLink(selectedDomain))}/${page.params.historyid ? encodeURIComponent(page.params.historyid) : ""}/${encodeURIComponent(page.params.subdomain!)}/${encodeURIComponent(page.params.serviceid!)}/checks`}
                    <ChecksSidebarContent
                        domain={data.domain}
                        checksBase={serviceChecksBase}
                        backHref={`/domains/${encodeURIComponent(domainLink(selectedDomain))}/${page.params.historyid ? encodeURIComponent(page.params.historyid) : ""}`}
                        serviceContext={page.data.zoneId
                            ? {
                                  zoneId: page.data.zoneId,
                                  subdomain: page.data.subdomain || page.params.subdomain!,
                                  serviceid: page.params.serviceid!,
                              }
                            : undefined}
                    />
                {:else if page.route.id && page.route.id.startsWith("/domains/[dn]/checks/[cname]")}
                    <ChecksSidebarContent
                        domain={data.domain}
                        checksBase={"/domains/" +
                            encodeURIComponent(domainLink(selectedDomain)) +
                            "/checks"}
                        backHref={"/domains/" +
                            encodeURIComponent(domainLink(selectedDomain)) +
                            "/checks"}
                    />
                {:else if page.route.id && (page.route.id.startsWith("/domains/[dn]/history") || page.route.id.startsWith("/domains/[dn]/logs") || page.route.id.startsWith("/domains/[dn]/[[historyid]]/export") || page.route.id == "/domains/[dn]/checks")}
                    <a
                        href="/domains/{encodeURIComponent(domainLink(selectedDomain))}"
                        class="sidebar-back d-flex align-items-center gap-1 mt-3 text-muted text-decoration-none fw-semibold"
                    >
                        <Icon name="chevron-left" />
                        {$t("zones.return-to")}
                    </a>
                {:else if page.route.id === "/domains/[dn]/[[historyid]]/[subdomain]/[serviceid]" || page.route.id.startsWith("/domains/[dn]/[[historyid]]/[subdomain]/[serviceid]/checks")}
                    <ServiceSidebar
                        origin={data.domain}
                        pageSuffix={page.route.id.startsWith(
                            "/domains/[dn]/[[historyid]]/[subdomain]/[serviceid]/checks",
                        )
                            ? "/checks"
                            : ""}
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

                <div class="flex-fill"></div>

                {#if !(data.domain.zone_history && $domains_idx[selectedDomain] && data.domain.id === $domains_idx[selectedDomain].id && selectedHistory)}
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
                {:else}
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
                (page.route.id == "/domains/[dn]/checks/[cname]/results/[rid]" ||
                    page.route.id ==
                        "/domains/[dn]/[[historyid]]/[subdomain]/[serviceid]/checks/[cname]/results/[rid]")}
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

<ModalDiffZone
    domain={data.domain}
    {selectedHistory}
    on:retrieveZoneDone={(ev) => retrieveZoneDone(ev.detail)}
/>

<ServiceDetailsOffcanvas domain={data.domain} {selectedHistory} />
