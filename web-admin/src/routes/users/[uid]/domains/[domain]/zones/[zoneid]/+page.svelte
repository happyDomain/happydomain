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
    import { page } from "$app/state";
    import { goto } from "$app/navigation";
    import {
        Accordion,
        AccordionItem,
        Alert,
        Button,
        Card,
        CardBody,
        CardHeader,
        Col,
        Container,
        Icon,
        Row,
        Spinner,
        Badge,
    } from "@sveltestrap/sveltestrap";

    import {
        getUsersByUidDomainsByDomainZonesByZoneid,
        deleteUsersByUidDomainsByDomainZonesByZoneid,
    } from "$lib/api-admin";
    import { toasts } from "$lib/stores/toasts";

    const uid = $derived(page.params.uid);
    const domainId = $derived(page.params.domain);
    const zoneid = $derived(page.params.zoneid);

    let zoneQ = $derived(
        getUsersByUidDomainsByDomainZonesByZoneid({
            path: { uid: uid!, domain: domainId!, zoneid: zoneid! },
        }),
    );

    let deleting = $state(false);

    async function handleDelete() {
        if (!confirm("Are you sure you want to delete this zone? This action cannot be undone.")) {
            return;
        }

        deleting = true;

        try {
            await deleteUsersByUidDomainsByDomainZonesByZoneid({
                path: { uid: uid!, domain: domainId!, zoneid: zoneid! },
            });

            toasts.addToast({
                message: `Zone has been deleted successfully`,
                type: "success",
                timeout: 5000,
            });

            goto(`/users/${uid}/domains/${domainId}`);
        } catch (error) {
            toasts.addErrorToast({
                message: "Failed to delete zone: " + error,
                timeout: 10000,
            });
        } finally {
            deleting = false;
        }
    }
</script>

<Container class="flex-fill my-5">
    <div class="d-flex align-items-center gap-1 mb-4">
        <Button color="link" href="/users/{uid}/domains/{domainId}" class="text-black">
            <Icon name="chevron-left"></Icon>
        </Button>
        <h1 class="display-5 mb-0">
            <Icon name="file-earmark-text"></Icon>
            Zone Details
        </h1>
    </div>

    {#await zoneQ}
        <div class="text-center my-5">
            <Spinner color="primary" />
            <p class="mt-3">Loading zone...</p>
        </div>
    {:then zoneR}
        {#if zoneR?.data}
            {@const zone = zoneR.data}
            <Row>
                <Col lg={12}>
                    <!-- Zone Information Card -->
                    <Card class="mb-4">
                        <CardHeader>
                            <div class="d-flex justify-content-between align-items-center">
                                <h5 class="mb-0">
                                    <Icon name="info-circle"></Icon>
                                    Zone Information
                                </h5>
                                <Badge color="primary">{zone.id}</Badge>
                            </div>
                        </CardHeader>
                        <CardBody>
                            <Row>
                                <Col md={6}>
                                    <dl class="row mb-0">
                                        <dt class="col-sm-5">Zone ID:</dt>
                                        <dd class="col-sm-7">
                                            <code class="text-break">{zone.id || "N/A"}</code>
                                        </dd>

                                        <dt class="col-sm-5">Author ID:</dt>
                                        <dd class="col-sm-7">
                                            {#if zone.id_author}
                                                <a href="/users/{zone.id_author}">
                                                    <code class="text-break">{zone.id_author}</code>
                                                </a>
                                            {:else}
                                                <span class="text-muted">N/A</span>
                                            {/if}
                                        </dd>

                                        <dt class="col-sm-5">Parent Zone:</dt>
                                        <dd class="col-sm-7">
                                            {#if zone.parent}
                                                <a
                                                    href="/users/{uid}/domains/{domainId}/zones/{zone.parent}"
                                                >
                                                    <code class="text-break">{zone.parent}</code>
                                                </a>
                                            {:else}
                                                <span class="text-muted">None</span>
                                            {/if}
                                        </dd>

                                        <dt class="col-sm-5">Default TTL:</dt>
                                        <dd class="col-sm-7">
                                            <Badge color="info">{zone.default_ttl || 3600}s</Badge>
                                        </dd>
                                    </dl>
                                </Col>
                                <Col md={6}>
                                    <dl class="row mb-0">
                                        <dt class="col-sm-5">Last Modified:</dt>
                                        <dd class="col-sm-7">
                                            {#if zone.last_modified}
                                                {new Date(zone.last_modified).toLocaleString()}
                                            {:else}
                                                <span class="text-muted">Unknown</span>
                                            {/if}
                                        </dd>

                                        <dt class="col-sm-5">Published:</dt>
                                        <dd class="col-sm-7">
                                            {#if zone.published}
                                                {new Date(zone.published).toLocaleString()}
                                            {:else}
                                                <Badge color="warning">
                                                    <Icon name="clock"></Icon>
                                                    Unpublished
                                                </Badge>
                                            {/if}
                                        </dd>

                                        {#if zone.commit_message}
                                            <dt class="col-sm-5">Commit Message:</dt>
                                            <dd class="col-sm-7">{zone.commit_message}</dd>
                                        {/if}

                                        {#if zone.commit_date}
                                            <dt class="col-sm-5">Commit Date:</dt>
                                            <dd class="col-sm-7">
                                                {new Date(zone.commit_date).toLocaleString()}
                                            </dd>
                                        {/if}
                                    </dl>
                                </Col>
                            </Row>
                        </CardBody>
                    </Card>

                    <!-- Services Card -->
                    <Card class="mb-4">
                        <CardHeader>
                            <div class="d-flex justify-content-between align-items-center">
                                <h5 class="mb-0">
                                    <Icon name="gear"></Icon>
                                    DNS Services
                                </h5>
                                <Badge color="secondary">
                                    {Object.keys(zone.services || {}).length} subdomains
                                </Badge>
                            </div>
                        </CardHeader>
                        <CardBody>
                            {#if zone.services && Object.keys(zone.services).length > 0}
                                {#each Object.entries(zone.services) as [subdomain, services]}
                                    <div class="mb-4">
                                        <h5 class="pb-2">
                                            {#if subdomain === ""}
                                                <span class="font-monospace">@</span>
                                                <small class="text-muted">root domain</small>
                                            {:else}
                                                <span class="font-monospace">{subdomain}</span>
                                            {/if}
                                        </h5>

                                        <Accordion class="mb-3">
                                            {#each services as service, idx}
                                                {@const headerText = `${service._svctype || "Unknown"}${service._comment ? " - " + service._comment : ""}${service._tmp_hint_nb ? ` (${service._tmp_hint_nb} record${service._tmp_hint_nb > 1 ? "s" : ""})` : ""}`}
                                                <AccordionItem header={headerText}>
                                                    <div class="small">
                                                        <div class="mb-1">
                                                            <strong>Service ID:</strong>
                                                            <code class="ms-2"
                                                                >{service._id || "N/A"}</code
                                                            >
                                                        </div>
                                                        {#if service._ttl}
                                                            <div class="mb-1">
                                                                <strong>TTL:</strong>
                                                                <Badge color="info" class="ms-2"
                                                                    >{service._ttl}s</Badge
                                                                >
                                                            </div>
                                                        {/if}
                                                        {#if service.Service}
                                                            <div class="mt-2">
                                                                <strong>DNS Records:</strong>
                                                                <pre
                                                                    class="bg-light p-2 mt-1 mb-0 rounded"><code
                                                                        >{JSON.stringify(
                                                                            service.Service,
                                                                            null,
                                                                            2,
                                                                        )}</code
                                                                    ></pre>
                                                            </div>
                                                        {/if}
                                                    </div>
                                                </AccordionItem>
                                            {/each}
                                        </Accordion>
                                    </div>
                                {/each}
                            {:else}
                                <p class="text-muted mb-0">No services configured for this zone.</p>
                            {/if}
                        </CardBody>
                    </Card>

                    <!-- Action Buttons -->
                    <div class="d-flex gap-2 mb-4">
                        <Button color="secondary" outline href="/users/{uid}/domains/{domainId}">
                            <Icon name="arrow-left"></Icon>
                            Back to Domain
                        </Button>
                        <Button
                            color="secondary"
                            outline
                            href="/users/{uid}/domains/{domainId}/zones"
                        >
                            <Icon name="clock-history"></Icon>
                            Back to History
                        </Button>
                        <div class="ms-auto">
                            <Button
                                color="danger"
                                outline
                                onclick={handleDelete}
                                disabled={deleting}
                            >
                                {#if deleting}
                                    <Spinner size="sm" class="me-2" />
                                {:else}
                                    <Icon name="trash" class="me-2"></Icon>
                                {/if}
                                Delete Zone
                            </Button>
                        </div>
                    </div>
                </Col>
            </Row>
        {:else}
            <Alert color="warning">
                <h4 class="alert-heading">No data available</h4>
                <p>The zone response did not contain any data.</p>
                <hr />
                <Button
                    type="button"
                    color="secondary"
                    outline
                    href="/users/{uid}/domains/{domainId}"
                >
                    <Icon name="arrow-left"></Icon>
                    Back to Domain
                </Button>
            </Alert>
        {/if}
    {:catch error}
        <Alert color="danger">
            <h4 class="alert-heading">Error loading zone</h4>
            <p>{error}</p>
            <hr />
            <Button type="button" color="secondary" outline href="/users/{uid}/domains/{domainId}">
                <Icon name="arrow-left"></Icon>
                Back to Domain
            </Button>
        </Alert>
    {/await}
</Container>
