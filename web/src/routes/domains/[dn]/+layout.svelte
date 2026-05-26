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

    import { Col, Container, Row } from "@sveltestrap/sveltestrap";

    import { deleteDomain as APIDeleteDomain } from "$lib/api/domains";
    import type { Domain } from "$lib/model/domain";
    import type { ZoneMeta } from "$lib/model/zone";
    import { domainLink, domains_idx, refreshDomains } from "$lib/stores/domains";
    import ModalDiffZone from "./ModalDiffZone.svelte";
    import ModalDomainDelete, { controls as ctrlDomainDelete } from "./ModalDomainDelete.svelte";
    import ModalDomainWhois from "./ModalDomainWhois.svelte";
    import ModalUploadZone from "./ModalUploadZone.svelte";
    import NewSubdomainPath from "./NewSubdomainPath.svelte";
    import ServiceDetailsOffcanvas from "./ServiceDetailsOffcanvas.svelte";
    import Sidebar from "./Sidebar.svelte";

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
                        ? page.route.id.startsWith("/domains/[dn]/checkers")
                            ? "/checkers"
                            : page.route.id.startsWith("/domains/[dn]/checks")
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
            class="py-2 sticky-top d-flex flex-column"
            style="background-color: #edf5f2; z-index: 0; max-height: 100vh; overflow-y: auto;"
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
                        "/domains/[dn]/[[historyid]]/[subdomain]/[serviceid]/checkers/[checkerId]/executions/[execId]")}
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

<ModalDomainWhois domain={data.domain.domain} />

<ModalDiffZone domain={data.domain} {selectedHistory} />

<ServiceDetailsOffcanvas domain={data.domain} {selectedHistory} />
