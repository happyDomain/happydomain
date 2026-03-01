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
        Badge,
        Card,
        CardBody,
        CardHeader,
        CardTitle,
        Col,
        ListGroup,
        ListGroupItem,
        Row,
    } from "@sveltestrap/sveltestrap";

    import type { DomainInfo } from "$lib/model/domaininfo";
    import { t } from "$lib/translations";

    interface Props {
        info: DomainInfo;
        domain: string;
    }

    let { info, domain }: Props = $props();

    function statusColor(code: string): string {
        const lc = code.toLowerCase();
        if (lc === "ok" || lc === "active") return "success";
        if (lc.includes("hold")) return "danger";
        if (lc.includes("prohibited") || lc.includes("pending")) return "info";
        return "secondary";
    }

    function expirationProgress(expiration?: string): number {
        if (!expiration) return 0;
        const days = daysUntilExpiration(expiration);
        if (days <= 0) return 0;
        return Math.min(100, Math.round((days / 365) * 100));
    }

    function expirationColor(expiration?: string): string {
        if (!expiration) return "secondary";
        const days = Math.round((new Date(expiration).getTime() - Date.now()) / 86400000);
        if (days < 0) return "danger";
        if (days < 30) return "danger";
        if (days < 90) return "warning";
        return "success";
    }

    function daysUntilExpiration(expiration?: string): number {
        if (!expiration) return 0;
        return Math.round((new Date(expiration).getTime() - Date.now()) / 86400000);
    }

    function formatDate(iso?: string): string {
        if (!iso) return "";
        return new Date(iso).toLocaleDateString(undefined, {
            year: "numeric",
            month: "long",
            day: "numeric",
        });
    }
</script>

<h2 class="display-7 fw-bold mt-3 mb-1 font-monospace">
    {info.name ?? domain}
</h2>

<!-- Status badges -->
{#if info.status && info.status.length > 0}
    <div class="mb-4">
        <p class="text-muted small mb-1">{$t("domaininfo.status")}</p>
        <div class="d-flex flex-wrap gap-2">
            {#each info.status as code}
                <Badge
                    color={statusColor(code)}
                    title={$t(`domaininfo.status-descriptions.${code}`) || code}
                >
                    {code}
                </Badge>
            {/each}
        </div>
        {#if info.status.length === 1}
            {@const desc = $t(
                `domaininfo.status-descriptions.${info.status[0]}`,
            )}
            {#if desc && !desc.startsWith("domaininfo.")}
                <p class="text-muted small mt-1 mb-0">{desc}</p>
            {/if}
        {/if}
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
                    {@const days = daysUntilExpiration(info.expiration)}
                    {@const color = expirationColor(info.expiration)}
                    {@const expiresLabel = days < 0 ? $t("domaininfo.expired", { days: Math.abs(days) }) : days === 0 ? $t("domaininfo.expires-today") : $t("domaininfo.expires-in", { days })}
                    <p class="fw-semibold mb-1">
                        {formatDate(info.expiration)}
                    </p>
                    <div
                        class="progress mb-1"
                        style="height: 6px;"
                        title={expiresLabel}
                    >
                        <div
                            class="progress-bar bg-{color}"
                            role="progressbar"
                            style="width: {expirationProgress(info.expiration)}%"
                        ></div>
                    </div>
                    <p class="text-{color} small mb-0">
                        {#if days < 0}
                            {$t("domaininfo.expired", { days: Math.abs(days) })}
                        {:else if days === 0}
                            {$t("domaininfo.expires-today")}
                        {:else}
                            {$t("domaininfo.expires-in", { days })}
                        {/if}
                    </p>
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
                            rel="noopener noreferrer"
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
                    {#each info.nameservers as ns}
                        <ListGroupItem class="font-monospace py-2"
                            >{ns}</ListGroupItem
                        >
                    {/each}
                </ListGroup>
            </Card>
        </Col>
    {/if}
</Row>
