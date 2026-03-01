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
    import { navigate } from "$lib/stores/config";
    import { untrack } from "svelte";
    import { preventDefault } from "svelte/legacy";

    import {
        Badge,
        Button,
        Card,
        CardBody,
        CardHeader,
        CardTitle,
        Col,
        Container,
        FormGroup,
        Input,
        ListGroup,
        ListGroupItem,
        Row,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { getDomainInfo } from "$lib/api/domaininfo";
    import type { DomainInfo } from "$lib/model/domaininfo";
    import { domains } from "$lib/stores/domains";
    import { t } from "$lib/translations";

    interface Props {
        data: { domain?: string };
    }

    let { data }: Props = $props();

    let domain = $derived(data.domain ?? "");
    let inputDomain = $state("");

    let error_response: string | null = $state(null);
    let not_found = $state(false);
    let request_pending = $state(false);
    let info: DomainInfo | null = $state(null);

    function fetchInfo(d: string) {
        if (!d) return;

        request_pending = true;
        error_response = null;
        not_found = false;
        info = null;

        getDomainInfo(d).then(
            (result) => {
                info = result;
                request_pending = false;
            },
            (error: unknown) => {
                const msg = error instanceof Error ? error.message : String(error);
                if (msg.toLowerCase().includes("not found") || msg.toLowerCase().includes("doesn't exist")) {
                    not_found = true;
                } else {
                    error_response = msg;
                }
                request_pending = false;
            },
        );
    }

    $effect(() => {
        if (domain) {
            untrack(() => {
                inputDomain = domain;
                fetchInfo(domain);
            });
        }
    });

    function submit() {
        if (!inputDomain) return;

        if (inputDomain === domain) {
            fetchInfo(inputDomain);
        } else {
            navigate("/whois/" + encodeURIComponent(inputDomain), { noScroll: true });
        }
    }

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

<svelte:head>
    <title>
        {$t("domaininfo.page-title")}
        {domain ? domain : ""}
        - happyDomain
    </title>
</svelte:head>

{#if domain}
    <Container fluid class="flex-fill d-flex flex-column">
        <Row class="flex-grow-1">
            <Col md={{ offset: 0, size: 4 }} class="bg-light pt-3 pb-5">
                <div class="sticky-top">
                    <div class="mb-4">
                        <h1 class="display-6 fw-bold">
                            {$t("domaininfo.page-title")}
                        </h1>
                    </div>
                    <form onsubmit={preventDefault(submit)}>
                        <FormGroup>
                            <label for="domain-input">
                                {$t("common.domain")}
                            </label>
                            <Input
                                id="domain-input"
                                class="font-monospace"
                                list="my-domains"
                                required
                                placeholder="example.com"
                                bind:value={inputDomain}
                            />
                            <div class="form-text">
                                {@html $t("domaininfo.domain-description", {
                                    domain: `<span class="font-monospace">example.com</span>`,
                                })}
                            </div>
                            <datalist id="my-domains">
                                {#if $domains}
                                    {#each $domains as dn (dn.id)}
                                        <option>{dn.domain}</option>
                                    {/each}
                                {/if}
                            </datalist>
                        </FormGroup>
                        <div class="mx-3 d-flex justify-content-end">
                            <Button type="submit" color="primary" disabled={request_pending}>
                                {#if request_pending}
                                    <Spinner size="sm" />
                                {/if}
                                {$t("domaininfo.lookup")}
                            </Button>
                        </div>
                    </form>
                </div>
            </Col>

            {#if request_pending}
                <Col md="8" class="pt-5 pb-5 d-flex align-items-center justify-content-center">
                    <div class="text-center text-muted">
                        <Spinner />
                        <p class="mt-3">{$t("common.spinning")}…</p>
                    </div>
                </Col>
            {:else if not_found}
                <Col md="8" class="pt-3 pb-5">
                    <h2 class="display-7 fw-bold mt-3">
                        <i class="bi bi-question-circle"></i>
                        {domain}
                    </h2>
                    <div class="card border-warning mt-3">
                        <div class="card-body">
                            <div class="d-flex align-items-center">
                                <i class="bi bi-exclamation-triangle text-warning fs-3 me-3"></i>
                                <p class="card-text mb-0">{$t("domaininfo.domain-not-found")}</p>
                            </div>
                        </div>
                    </div>
                </Col>
            {:else if error_response !== null}
                <Col md="8" class="pt-3 pb-5">
                    <h2 class="display-7 fw-bold mt-3">
                        <i class="bi bi-exclamation-triangle"></i>
                        {$t("domaininfo.error")}
                    </h2>
                    <div class="card border-danger mt-3">
                        <div class="card-body">
                            <div class="d-flex align-items-center">
                                <i class="bi bi-x-circle text-danger fs-3 me-3"></i>
                                <p class="card-text mb-0">{error_response}</p>
                            </div>
                        </div>
                    </div>
                </Col>
            {:else if info !== null}
                <Col md="8" class="pt-3 pb-5">
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
                </Col>
            {/if}
        </Row>
    </Container>
{:else}
    <div class="my-5 container flex-fill">
        <div class="text-center">
            <h1 class="display-6 fw-bold">
                <i class="bi bi-info-circle"></i>
                {$t("domaininfo.page-title")}
            </h1>
            <p class="lead mt-1">
                {$t("domaininfo.page-description")}
            </p>
        </div>
        <Row class="justify-content-center mt-4">
            <Col md="10" lg="8">
                <div class="card rounded-4 p-2">
                    <div class="card-body">
                        <form onsubmit={preventDefault(submit)}>
                            <FormGroup>
                                <label for="domain-input-landing">
                                    {$t("common.domain")}
                                </label>
                                <Input
                                    id="domain-input-landing"
                                    class="font-monospace"
                                    list="my-domains-landing"
                                    required
                                    placeholder="example.com"
                                    bind:value={inputDomain}
                                />
                                <div class="form-text">
                                    {@html $t("domaininfo.domain-description", {
                                        domain: `<span class="font-monospace">example.com</span>`,
                                    })}
                                </div>
                                <datalist id="my-domains-landing">
                                    {#if $domains}
                                        {#each $domains as dn (dn.id)}
                                            <option>{dn.domain}</option>
                                        {/each}
                                    {/if}
                                </datalist>
                            </FormGroup>
                            <div class="d-flex justify-content-end">
                                <Button type="submit" color="primary" disabled={request_pending}>
                                    {#if request_pending}
                                        <Spinner size="sm" />
                                    {/if}
                                    {$t("domaininfo.lookup")}
                                </Button>
                            </div>
                        </form>
                    </div>
                </div>
            </Col>
        </Row>
    </div>
{/if}
