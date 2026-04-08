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
    import { onDestroy, onMount } from "svelte";
    import {
        Alert,
        Badge,
        Button,
        Card,
        CardFooter,
        CardHeader,
        Col,
        Collapse,
        Container,
        Icon,
        ListGroup,
        ListGroupItem,
        Row,
    } from "@sveltestrap/sveltestrap";

    import { fetchMetrics, firstLabel, singleValue, sumValues, type Metrics } from "$lib/metrics";
    import { formatBytes, formatDuration } from "$lib/utils";

    // formatDuration in $lib/utils takes nanoseconds and may emit decimals;
    // metrics expose seconds and we want whole units, so floor to a unit
    // boundary before delegating.
    function fmtSeconds(s: number | undefined): string {
        if (s === undefined || !Number.isFinite(s)) return formatDuration(undefined);
        const sec = Math.floor(s);
        let unitNs: number;
        if (sec < 60) unitNs = 1e9;
        else if (sec < 3600) unitNs = 60 * 1e9;
        else if (sec < 86400) unitNs = 3600 * 1e9;
        else unitNs = 86400 * 1e9;
        return formatDuration(Math.floor((sec * 1e9) / unitNs) * unitNs);
    }
    import DatabaseBackupCard from "./DatabaseBackupCard.svelte";
    import TidyCard from "./TidyCard.svelte";

    let metrics: Metrics | undefined = $state();
    let metricsError: string | undefined = $state();
    let lastUpdated: Date | undefined = $state();
    let isRefreshing = $state(false);
    let showMore = $state(false);
    let now = $state(Date.now() / 1000);
    let refreshTimer: ReturnType<typeof setInterval> | undefined;
    let tickTimer: ReturnType<typeof setInterval> | undefined;

    async function refresh() {
        isRefreshing = true;
        try {
            metrics = await fetchMetrics();
            metricsError = undefined;
            lastUpdated = new Date();
        } catch (err) {
            metricsError = err instanceof Error ? err.message : String(err);
        } finally {
            isRefreshing = false;
        }
    }

    onMount(() => {
        refresh();
        refreshTimer = setInterval(refresh, 15000);
        tickTimer = setInterval(() => (now = Date.now() / 1000), 1000);
    });

    onDestroy(() => {
        if (refreshTimer) clearInterval(refreshTimer);
        if (tickTimer) clearInterval(tickTimer);
    });

    let totalUsers = $derived(singleValue(metrics ?? {}, "happydomain_registered_users"));
    let totalDomains = $derived(singleValue(metrics ?? {}, "happydomain_domains"));
    let totalProviders = $derived(singleValue(metrics ?? {}, "happydomain_providers"));
    let totalZones = $derived(singleValue(metrics ?? {}, "happydomain_zones"));
    let schedulerQueue = $derived(singleValue(metrics ?? {}, "happydomain_scheduler_queue_depth"));
    let schedulerWorkers = $derived(
        singleValue(metrics ?? {}, "happydomain_scheduler_active_workers"),
    );
    let httpInFlight = $derived(singleValue(metrics ?? {}, "happydomain_http_requests_in_flight"));
    let buildVersion = $derived(firstLabel(metrics ?? {}, "happydomain_build_info", "version"));

    let httpRequestsTotal = $derived(sumValues(metrics ?? {}, "happydomain_http_requests_total"));
    let checksTotal = $derived(sumValues(metrics ?? {}, "happydomain_scheduler_checks_total"));
    let providerCallsTotal = $derived(
        sumValues(metrics ?? {}, "happydomain_provider_api_calls_total"),
    );
    let storageOpsTotal = $derived(
        sumValues(metrics ?? {}, "happydomain_storage_operations_total"),
    );
    let storageStatsErrors = $derived(
        sumValues(metrics ?? {}, "happydomain_storage_stats_errors_total"),
    );
    let goRoutines = $derived(singleValue(metrics ?? {}, "go_goroutines"));
    let goThreads = $derived(singleValue(metrics ?? {}, "go_threads"));
    let goMemAlloc = $derived(singleValue(metrics ?? {}, "go_memstats_alloc_bytes"));
    let processRSS = $derived(singleValue(metrics ?? {}, "process_resident_memory_bytes"));
    let processCPU = $derived(singleValue(metrics ?? {}, "process_cpu_seconds_total"));
    let processOpenFDs = $derived(singleValue(metrics ?? {}, "process_open_fds"));
    let processStart = $derived(singleValue(metrics ?? {}, "process_start_time_seconds"));
    let uptime = $derived(processStart === undefined ? undefined : now - processStart);

    function fmt(v: number | undefined): string {
        if (v === undefined || !Number.isFinite(v)) return "—";
        return Math.round(v).toLocaleString();
    }

    let checksFailed = $derived.by(() => {
        const samples = metrics?.["happydomain_scheduler_checks_total"];
        if (!samples) return undefined;
        return samples
            .filter((s) => {
                const st = s.labels["status"];
                return st && st !== "ok" && st !== "success";
            })
            .reduce((acc, s) => acc + s.value, 0);
    });
</script>

{#snippet tile(label: string, value: string, sub: string | null, icon: string, color: string)}
    <Col>
        <Card body class="h-100 border-0 shadow-sm">
            <div class="d-flex justify-content-between align-items-start mb-2">
                <h6 class="text-muted text-uppercase small mb-0">{label}</h6>
                <i class="bi bi-{icon} text-{color}" style="font-size: 1.25rem;"></i>
            </div>
            <div class="fs-2 fw-semibold font-monospace">{value}</div>
            {#if sub}<div class="small text-muted">{sub}</div>{/if}
        </Card>
    </Col>
{/snippet}

{#snippet row(label: string, value: string, badge: string | null)}
    <ListGroupItem class="d-flex justify-content-between align-items-center bg-transparent">
        <span class="text-muted small">{label}</span>
        <span class="font-monospace">
            {value}
            {#if badge}<Badge color="warning" class="text-dark ms-2">{badge}</Badge>{/if}
        </span>
    </ListGroupItem>
{/snippet}

<Container class="flex-fill my-5">
    <div class="d-flex justify-content-between align-items-start flex-wrap gap-3 mb-4">
        <div>
            <h1 class="display-5 mb-1">
                <Icon name="speedometer2" class="text-primary"></Icon>
                Admin Dashboard
            </h1>
            <p class="text-muted mb-0">
                Live telemetry from <code>/metrics</code>, refreshed every 15s.
            </p>
        </div>
        <Button type="button" color="secondary" outline disabled={isRefreshing} on:click={refresh}>
            <Icon name="arrow-repeat" class="me-1 {isRefreshing ? 'spin' : ''}"></Icon>
            Refresh
        </Button>
    </div>

    <div class="d-flex flex-wrap gap-2 mb-4">
        {#if buildVersion}
            <Badge class="bg-secondary-subtle text-secondary-emphasis border">
                <Icon name="tag" class="me-1"></Icon>v{buildVersion}
            </Badge>
        {/if}
        <Badge class="bg-secondary-subtle text-secondary-emphasis border tnum">
            <i class="bi bi-clock me-1"></i>uptime {fmtSeconds(uptime)}
        </Badge>
        <Badge class="bg-success-subtle text-success-emphasis border tnum">
            <Icon name="broadcast" class="me-1"></Icon>
            {lastUpdated ? `updated ${lastUpdated.toLocaleTimeString()}` : "connecting…"}
        </Badge>
    </div>

    {#if metricsError}
        <Alert color="warning" class="d-flex align-items-center">
            <Icon name="exclamation-triangle" class="me-2"></Icon>
            <div>Failed to load metrics: {metricsError}</div>
        </Alert>
    {/if}

    <h2 class="h5 text-muted text-uppercase small fw-bold mt-4 mb-3">Inventory</h2>
    <div class="row row-cols-2 row-cols-lg-4 g-3 mb-4">
        {@render tile("Users", fmt(totalUsers), "registered", "people-fill", "primary")}
        {@render tile("Domains", fmt(totalDomains), "managed", "globe2", "primary")}
        {@render tile(
            "Providers",
            fmt(totalProviders),
            "DNS backends",
            "hdd-network-fill",
            "primary",
        )}
        {@render tile("Zone snapshots", fmt(totalZones), "stored", "clock-history", "primary")}
    </div>

    <h2 class="h5 text-muted text-uppercase small fw-bold mt-4 mb-3">Runtime</h2>
    <div class="row row-cols-2 row-cols-lg-4 g-3 mb-4">
        {@render tile("Checker queue", fmt(schedulerQueue), "queued", "list-task", "info")}
        {@render tile("Active workers", fmt(schedulerWorkers), "running", "cpu", "info")}
        {@render tile("HTTP in flight", fmt(httpInFlight), "serving", "arrow-left-right", "info")}
        {@render tile("Memory RSS", formatBytes(processRSS), "resident", "memory", "info")}
    </div>

    <div class="text-center mb-3">
        <Button
            color="link"
            class="text-decoration-none"
            onclick={() => (showMore = !showMore)}
            aria-expanded={showMore}
        >
            {showMore ? "Hide" : "Show"} detailed metrics
            <i class="bi bi-chevron-{showMore ? 'up' : 'down'} ms-1"></i>
        </Button>
    </div>

    <Collapse isOpen={showMore}>
        <Card class="border-0 shadow-sm mb-4">
            <CardHeader class="bg-transparent">
                <h3 class="h6 mb-0">Detailed metrics</h3>
            </CardHeader>
            <Row class="g-0">
                <Col md={6}>
                    <div class="px-3 pt-2 small text-uppercase text-muted fw-bold">
                        Traffic &amp; work
                    </div>
                    <ListGroup flush>
                        {@render row("HTTP requests served", fmt(httpRequestsTotal), null)}
                        {@render row(
                            "Checks executed",
                            fmt(checksTotal),
                            checksFailed !== undefined && checksFailed > 0
                                ? `${fmt(checksFailed)} failed`
                                : null,
                        )}
                        {@render row("Provider API calls", fmt(providerCallsTotal), null)}
                        {@render row("Storage operations", fmt(storageOpsTotal), null)}
                        {@render row(
                            "Storage stats errors",
                            fmt(storageStatsErrors),
                            storageStatsErrors && storageStatsErrors > 0 ? "warn" : null,
                        )}
                    </ListGroup>
                </Col>
                <Col md={6}>
                    <div class="px-3 pt-2 small text-uppercase text-muted fw-bold">
                        Runtime &amp; process
                    </div>
                    <ListGroup flush>
                        {@render row("CPU time", fmtSeconds(processCPU), null)}
                        {@render row("Heap allocated", formatBytes(goMemAlloc), null)}
                        {@render row("Goroutines", fmt(goRoutines), null)}
                        {@render row("OS threads", fmt(goThreads), null)}
                        {@render row("Open file descriptors", fmt(processOpenFDs), null)}
                    </ListGroup>
                </Col>
            </Row>
            <CardFooter class="bg-transparent text-muted small">
                Source: <a href="/metrics" target="_blank" rel="noopener">/metrics</a> (Prometheus).
            </CardFooter>
        </Card>
    </Collapse>

    <TidyCard class="my-4" />

    <DatabaseBackupCard class="my-4" />
</Container>

<style>
    .tnum {
        font-variant-numeric: tabular-nums;
    }
    .spin {
        display: inline-block;
        animation: spin 0.8s linear infinite;
    }
    @keyframes spin {
        to {
            transform: rotate(360deg);
        }
    }
</style>
