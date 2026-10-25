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
    import {
        Card,
        CardBody,
        CardHeader,
        CardTitle,
        Col,
        ListGroup,
        ListGroupItem,
        Row,
    } from "@sveltestrap/sveltestrap";

    import DomainStatusBadges from "$lib/components/DomainStatusBadges.svelte";
    import type { DomainInfo } from "$lib/model/domaininfo";
    import { t } from "$lib/translations";
    import {
        daysUntil,
        expirationColor,
        expirationLabel,
        expirationProgress,
        formatDate,
    } from "$lib/utils/domaininfo";

    interface Props {
        info: DomainInfo;
        domain: string;
    }

    let { info, domain }: Props = $props();
</script>

<h2 class="display-7 fw-bold mt-3 mb-1 font-monospace">
    {info.name ?? domain}
</h2>

<!-- Status badges -->
{#if info.status && info.status.length > 0}
    <div class="mb-4">
        <p class="text-muted small mb-1">{$t("domaininfo.status")}</p>
        <DomainStatusBadges status={info.status} />
    </div>
{/if}

<!-- Dates card -->
<div class="card mb-4">
    <div class="card-body">
        <Row>
            <Col sm="6" class="mb-3 mb-sm-0">
                <p class="text-muted small mb-1">
                    <i class="bi bi-calendar-check me-1"></i>
                    {$t("domaininfo.creation-date")}
                </p>
                {#if info.creation}
                    <p class="fw-semibold mb-0">{formatDate(info.creation)}</p>
                {:else}
                    <p class="text-muted mb-0">
                        {$t("domaininfo.no-creation")}
                    </p>
                {/if}
            </Col>
            <Col sm="6">
                <p class="text-muted small mb-1">
                    <i class="bi bi-calendar-x me-1"></i>
                    {$t("domaininfo.expiration-date")}
                </p>
                {#if info.expiration}
                    {@const days = daysUntil(info.expiration)}
                    {@const color = expirationColor(info.expiration)}
                    {@const expiresLabel = $t(...expirationLabel(days))}
                    <p class="fw-semibold mb-1">
                        {formatDate(info.expiration)}
                    </p>
                    <div class="progress mb-1" style="height: 6px;" title={expiresLabel}>
                        <div
                            class="progress-bar bg-{color}"
                            role="progressbar"
                            style="width: {expirationProgress(info.expiration)}%"
                        ></div>
                    </div>
                    <p class="text-{color} small mb-0">{expiresLabel}</p>
                {:else}
                    <p class="text-muted mb-0">
                        {$t("domaininfo.no-expiration")}
                    </p>
                {/if}
            </Col>
        </Row>
    </div>
</div>

<Row>
    <Col md={6}>
        <!-- Registrar card -->
        <Card class="mb-4">
            <CardHeader>
                <CardTitle class="h6 mb-0 fw-bold">
                    <i class="bi bi-building me-1"></i>
                    {$t("domaininfo.registrar")}
                </CardTitle>
            </CardHeader>
            <CardBody>
                {#if info.registrar}
                    <p class="fw-semibold mb-1">{info.registrar}</p>
                    {#if info.registrar_url}
                        <a
                            href={info.registrar_url}
                            target="_blank"
                            rel="external noopener noreferrer"
                            class="btn btn-outline-secondary btn-sm"
                        >
                            <i class="bi bi-box-arrow-up-right me-1"></i>
                            {$t("domaininfo.registrar-url")}
                        </a>
                    {/if}
                {:else}
                    <p class="text-muted mb-0">
                        {$t("domaininfo.no-registrar")}
                    </p>
                {/if}
            </CardBody>
        </Card>
    </Col>

    <!-- Nameservers card -->
    {#if info.nameservers && info.nameservers.length > 0}
        <Col md={6}>
            <Card class="mb-4">
                <CardHeader>
                    <CardTitle class="h6 mb-0 fw-bold">
                        <i class="bi bi-hdd-network me-1"></i>
                        {$t("domaininfo.nameservers")}
                    </CardTitle>
                </CardHeader>
                <ListGroup flush>
                    {#each info.nameservers as ns, i (i)}
                        <ListGroupItem class="font-monospace py-2">{ns}</ListGroupItem>
                    {/each}
                </ListGroup>
            </Card>
        </Col>
    {/if}
</Row>
