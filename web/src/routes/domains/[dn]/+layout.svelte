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
    import { resolve } from "$app/paths";
    import { goto, invalidateAll } from "$app/navigation";
    import { page } from "$app/state";

    import { Col, Container, Row } from "@sveltestrap/sveltestrap";

    import { deleteDomain as APIDeleteDomain } from "$lib/api/domains";
    import type { Domain } from "$lib/model/domain";
    import type { Zone, ZoneMeta } from "$lib/model/zone";
    import { domainLink, domains_idx, refreshDomains } from "$lib/stores/domains";
    import { getZone, thisZone } from "$lib/stores/thiszone";
    import { invalidateZoneDiff } from "$lib/stores/zonediff";
    import { refreshDomainChecks } from "$lib/stores/domainchecks";
    import ModalDiffZone from "./ModalDiffZone.svelte";
    import ModalDomainDelete, { controls as ctrlDomainDelete } from "./ModalDomainDelete.svelte";
    import ModalDomainWhois from "./ModalDomainWhois.svelte";
    import ModalUploadZone from "./ModalUploadZone.svelte";
    import ModalApplyTemplate from "./ModalApplyTemplate.svelte";
    import ModalSaveAsTemplate from "./ModalSaveAsTemplate.svelte";
    import NewSubdomainPath from "$lib/components/zones/NewSubdomainPath.svelte";
    import NewServicePath from "$lib/components/services/NewServicePath.svelte";
    import { domainZoneEditor, provideZoneEditor } from "$lib/components/zones/editor";
    import ServiceDetailsOffcanvas from "$lib/components/zones/ServiceDetailsOffcanvas.svelte";
    import Sidebar from "./Sidebar.svelte";

    interface Props {
        data: { domain: Domain };
        children?: import("svelte").Snippet;
    }

    let { data, children }: Props = $props();

    // Under a version of the history, the zone is only shown.
    provideZoneEditor(
        domainZoneEditor(
            () => data.domain,
            () => page.params.zid,
        ),
    );

    let selectedDomain = $derived(data.domain.id);
    function domainChange(dn: string) {
        if (dn != data.domain.id) {
            goto(sameSectionOf(encodeURIComponent(domainLink(dn))));
        }
        if (selectedDomain != dn) {
            selectedDomain = dn;
        }
    }

    let selectedHistory: string | undefined = $derived(page.data.history);

    // The checks of the current zone, shared by the dashboard and the
    // sidebars: loaded with each zone, and again on coming back to the
    // dashboard, as they may have run meanwhile.
    let checkedZone: Zone | null = null;
    $effect(() => {
        const zone = $thisZone;
        const onDashboard = page.route.id === "/domains/[dn]";
        if (!zone || page.params.zid || zone.id !== data.domain.zone_history?.[0]) return;
        if (zone !== checkedZone || onDashboard) {
            checkedZone = zone;
            refreshDomainChecks(data.domain.id, zone);
        }
    });

    // The zone retrieved is the latest of the history: the current one.
    function retrieveZoneDone(_zm: ZoneMeta): void {
        refreshDomains().then(() => {
            invalidateAll();
        });
    }

    // Each publication adds a snapshot to the history: refresh what the
    // current page shows of it.
    function zoneApplied(zm: ZoneMeta): void {
        refreshDomains().then(() => {
            if (page.route.id?.startsWith("/domains/[dn]/history")) {
                goto(
                    resolve("/domains/[dn]/history/[zid]", {
                        dn: encodeURIComponent(domainLink(selectedDomain)),
                        zid: encodeURIComponent(zm.id),
                    }),
                    { invalidateAll: true },
                );
            } else if ($thisZone) {
                // The diff summaries compare against the zone reloaded here.
                getZone(data.domain, $thisZone.id).then(() => invalidateZoneDiff());
            }
        });
    }

    let deleteInProgress = $state(false);
    function detachDomain(): void {
        deleteInProgress = true;
        APIDeleteDomain($domains_idx[selectedDomain].id).then(
            () => {
                refreshDomains().then(
                    () => {
                        deleteInProgress = false;
                        goto(resolve("/domains"));
                    },
                    () => {
                        deleteInProgress = false;
                        goto(resolve("/domains"));
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

    // Switching domains keeps the user on the same section.
    function sameSectionOf(dn: string) {
        const id = page.route.id ?? "";
        if (id.startsWith("/domains/[dn]/checkers"))
            return resolve("/domains/[dn]/checkers", { dn });
        if (id.startsWith("/domains/[dn]/logs")) return resolve("/domains/[dn]/logs", { dn });
        if (id.startsWith("/domains/[dn]/history")) return resolve("/domains/[dn]/history", { dn });
        if (id.startsWith("/domains/[dn]/export")) return resolve("/domains/[dn]/export", { dn });
        if (id.startsWith("/domains/[dn]/zone")) return resolve("/domains/[dn]/zone", { dn });
        return resolve("/domains/[dn]", { dn });
    }
</script>

<svelte:head>
    <title>{data.domain.domain} - HappyDomain</title>
</svelte:head>

<Container fluid class="d-flex flex-column flex-fill">
    <Row class="flex-fill">
        <Col
            sm={4}
            md={3}
            class="py-2 sticky-top d-flex flex-column"
            style="background-color: var(--bs-secondary-bg); z-index: 0; max-height: 100vh; overflow-y: auto;"
        >
            <Sidebar
                domain={data.domain}
                bind:selectedDomain
                {selectedHistory}
                {deleteInProgress}
                ondetachClick={() => ctrlDomainDelete.Open()}
                onretrieveZoneDone={retrieveZoneDone}
            />
        </Col>

        <div
            class="col-sm-8 col-md-9 d-flex"
            class:p-0={page.route &&
                (page.route.id == "/domains/[dn]/checkers/[checkerId]/executions/[execId]" ||
                    page.route.id ==
                        "/domains/[dn]/zone/[subdomain]/[serviceid]/checkers/[checkerId]/executions/[execId]")}
        >
            {@render children?.()}
        </div>
    </Row>
</Container>

<NewSubdomainPath origin={data.domain} />

<!-- Services are added to the current zone, from the dashboard as from the zone. -->
{#if $thisZone && !page.params.zid && $thisZone.id === data.domain.zone_history?.[0]}
    <NewServicePath origin={data.domain} zone={$thisZone} historyId={$thisZone.id} />
{/if}

<ModalUploadZone domain={data.domain} on:retrieveZoneDone={(ev) => retrieveZoneDone(ev.detail)} />

<ModalSaveAsTemplate domain={data.domain} {selectedHistory} />

<ModalApplyTemplate domain={data.domain} {selectedHistory} onapplied={retrieveZoneDone} />

<ModalDomainDelete on:detachDomain={detachDomain} />

<ModalDomainWhois domain={data.domain.domain} />

<ModalDiffZone domain={data.domain} {selectedHistory} onapplied={zoneApplied} />

<ServiceDetailsOffcanvas domain={data.domain} {selectedHistory} />
