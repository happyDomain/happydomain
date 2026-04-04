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
    import { onMount } from "svelte";
    import {
        Badge,
        Button,
        Card,
        CardBody,
        CardHeader,
        Col,
        Container,
        Icon,
        Row,
        Spinner,
        Table,
    } from "@sveltestrap/sveltestrap";

    import {
        getScheduler,
        postSchedulerEnable,
        postSchedulerDisable,
        postSchedulerRescheduleUpcoming,
    } from "$lib/api-admin";
    import type { CheckerSchedulerStatus } from "$lib/api-admin";
    import { formatDuration, formatRelative } from "$lib/utils/datetime";

    let status = $state<CheckerSchedulerStatus | null>(null);
    let loading = $state(true);
    let toggling = $state(false);
    let rescheduling = $state(false);
    let error = $state<string | null>(null);

    async function fetchStatus() {
        loading = true;
        error = null;
        try {
            const { data, error: err } = await getScheduler();
            if (err) throw new Error(String(err));
            status = data ?? null;
        } catch (e: any) {
            error = e.message ?? "Unknown error";
        } finally {
            loading = false;
        }
    }

    async function toggleScheduler() {
        if (!status) return;
        toggling = true;
        error = null;
        try {
            const fn = status.running ? postSchedulerDisable : postSchedulerEnable;
            const { data, error: err } = await fn();
            if (err) throw new Error(String(err));
            status = data ?? null;
        } catch (e: any) {
            error = e.message ?? "Unknown error";
        } finally {
            toggling = false;
        }
    }

    async function rebuildQueue() {
        rescheduling = true;
        error = null;
        try {
            const { error: err } = await postSchedulerRescheduleUpcoming();
            if (err) throw new Error(String(err));
            await fetchStatus();
        } catch (e: any) {
            error = e.message ?? "Unknown error";
        } finally {
            rescheduling = false;
        }
    }

    onMount(fetchStatus);
</script>

<Container class="flex-fill my-5">
    <Row class="mb-4">
        <Col>
            <h1 class="display-5">
                <Icon name="clock-history"></Icon>
                Scheduler
            </h1>
            <p class="text-muted lead">Monitor and control the checker scheduler</p>
        </Col>
    </Row>

    {#if error}
        <Card color="danger" body class="mb-4">
            <Icon name="exclamation-triangle-fill"></Icon>
            {error}
        </Card>
    {/if}

    {#if loading}
        <div class="d-flex align-items-center gap-2">
            <Spinner size="sm" />
            <span>Loading scheduler status...</span>
        </div>
    {:else if status}
        <Card class="mb-4">
            <CardHeader>
                <div class="d-flex justify-content-between align-items-center">
                    <span>
                        <Icon name="info-circle-fill"></Icon>
                        Scheduler Status
                    </span>
                    <div class="d-flex gap-2">
                        <Button size="sm" color="secondary" outline onclick={fetchStatus}>
                            <Icon name="arrow-clockwise"></Icon> Refresh
                        </Button>
                        <Button
                            size="sm"
                            color={status.running ? "warning" : "success"}
                            disabled={toggling}
                            onclick={toggleScheduler}
                        >
                            {#if toggling}
                                <Spinner size="sm" />
                            {:else if status.running}
                                <Icon name="stop-fill"></Icon> Stop
                            {:else}
                                <Icon name="play-fill"></Icon> Start
                            {/if}
                        </Button>
                        <Button
                            size="sm"
                            color="primary"
                            outline
                            disabled={rescheduling}
                            onclick={rebuildQueue}
                        >
                            {#if rescheduling}
                                <Spinner size="sm" />
                            {:else}
                                <Icon name="calendar2-check"></Icon> Rebuild queue
                            {/if}
                        </Button>
                    </div>
                </div>
            </CardHeader>
            <CardBody>
                <div class="d-flex gap-4 align-items-center">
                    <div>
                        <small class="text-muted d-block">Status</small>
                        {#if status.running}
                            <Badge color="success"><Icon name="play-fill"></Icon> Running</Badge>
                        {:else}
                            <Badge color="secondary"><Icon name="stop-fill"></Icon> Stopped</Badge>
                        {/if}
                    </div>
                    <div>
                        <small class="text-muted d-block">Jobs in queue</small>
                        <strong>{status.job_count ?? 0}</strong>
                    </div>
                </div>
            </CardBody>
        </Card>

        <Card>
            <CardHeader>
                <Icon name="list-ol"></Icon>
                Next scheduled jobs
                <Badge color="secondary" class="ms-2">{status.next_jobs?.length ?? 0}</Badge>
            </CardHeader>
            <CardBody class="p-0">
                <div class="table-responsive">
                    <Table hover class="mb-0">
                        <thead>
                            <tr>
                                <th>Checker</th>
                                <th>Target</th>
                                <th>Interval</th>
                                <th>Next run</th>
                            </tr>
                        </thead>
                        <tbody>
                            {#if !status.next_jobs || status.next_jobs.length === 0}
                                <tr>
                                    <td colspan="4" class="text-center text-muted py-3">
                                        No jobs scheduled
                                    </td>
                                </tr>
                            {:else}
                                {#each status.next_jobs as job}
                                    <tr>
                                        <td>
                                            <code>{job.checkerID ?? "—"}</code>
                                        </td>
                                        <td>
                                            {#if job.target?.domainId}
                                                <Badge
                                                    href={"/domains/" + job.target?.domainId}
                                                    color="info"
                                                    class="me-1"
                                                >
                                                    domain
                                                </Badge>
                                            {/if}
                                            {#if job.target?.serviceId}
                                                <Badge
                                                    href={"/service/" + job.target?.serviceId}
                                                    color="warning"
                                                    class="me-1"
                                                >
                                                    service
                                                </Badge>
                                            {/if}
                                            {#if job.target?.userId}
                                                <Badge
                                                    href={"/users/" + job.target?.userId}
                                                    color="secondary"
                                                    class="me-1"
                                                >
                                                    user
                                                </Badge>
                                            {/if}
                                            {#if !job.target?.domainId && !job.target?.serviceId && !job.target?.userId}
                                                <span class="text-muted">—</span>
                                            {/if}
                                        </td>
                                        <td>{formatDuration(job.interval)}</td>
                                        <td>
                                            <span title={job.nextRun}
                                                >{formatRelative(job.nextRun)}</span
                                            >
                                        </td>
                                    </tr>
                                {/each}
                            {/if}
                        </tbody>
                    </Table>
                </div>
            </CardBody>
        </Card>
    {/if}
</Container>
