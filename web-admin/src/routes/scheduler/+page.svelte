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

    import { toasts } from "$lib/stores/toasts";
    import {
        getScheduler,
        postSchedulerDisable,
        postSchedulerEnable,
        postSchedulerRescheduleUpcoming,
    } from "$lib/api-admin/sdk.gen";

    interface CheckerSchedule {
        id: string;
        checker_name: string;
        owner_id: string;
        target_type: number;
        target_id: string;
        interval: number;
        enabled: boolean;
        last_run?: string;
        next_run: string;
    }

    interface SchedulerStatus {
        config_enabled: boolean;
        runtime_enabled: boolean;
        running: boolean;
        worker_count: number;
        queue_size: number;
        active_count: number;
        next_schedules: CheckerSchedule[] | null;
    }

    let status = $state<SchedulerStatus | null>(null);
    let loading = $state(true);
    let actionInProgress = $state(false);
    let rescheduleInProgress = $state(false);
    let error = $state<string | null>(null);

    async function fetchStatus() {
        loading = true;
        error = null;
        try {
            const { data, error: err } = await getScheduler();
            if (err) throw new Error(String(err));
            status = data as SchedulerStatus;
        } catch (e: any) {
            error = e.message ?? "Unknown error";
        } finally {
            loading = false;
        }
    }

    async function setEnabled(enabled: boolean) {
        actionInProgress = true;
        const action = enabled ? "enable" : "disable";
        try {
            const { data, error: err } = await (enabled
                ? postSchedulerEnable()
                : postSchedulerDisable());
            if (err) {
                toasts.addErrorToast({ message: `Failed to ${action} scheduler: ${err}` });
                return;
            }
            status = data as SchedulerStatus;
            toasts.addToast({ message: `Scheduler ${action}d successfully`, color: "success" });
        } catch (e: any) {
            toasts.addErrorToast({ message: e.message ?? `Failed to ${action} scheduler` });
        } finally {
            actionInProgress = false;
        }
    }

    async function rescheduleUpcoming() {
        rescheduleInProgress = true;
        try {
            const { data, error: err } = await postSchedulerRescheduleUpcoming();
            if (err) {
                toasts.addErrorToast({ message: `Failed to reschedule: ${err}` });
                return;
            }
            toasts.addToast({
                message: `Rescheduled ${(data as any).rescheduled} schedule(s) successfully`,
                color: "success",
            });
            await fetchStatus();
        } catch (e: any) {
            toasts.addErrorToast({ message: e.message ?? "Failed to reschedule upcoming checks" });
        } finally {
            rescheduleInProgress = false;
        }
    }

    function formatDuration(ns: number): string {
        const seconds = ns / 1e9;
        if (seconds < 60) return `${Math.round(seconds)}s`;
        const minutes = seconds / 60;
        if (minutes < 60) return `${Math.round(minutes)}m`;
        const hours = minutes / 60;
        if (hours < 24) return `${Math.round(hours)}h`;
        return `${Math.round(hours / 24)}d`;
    }

    function targetTypeName(t: number): string {
        const names: Record<number, string> = {
            0: "instance",
            1: "user",
            2: "domain",
            3: "zone",
            4: "service",
            5: "ondemand",
        };
        return names[t] ?? "unknown";
    }

    onMount(fetchStatus);
</script>

<Container class="flex-fill my-5">
    <Row class="mb-4">
        <Col>
            <h1 class="display-5">
                <Icon name="clock-history"></Icon>
                Test Scheduler
            </h1>
            <p class="text-muted lead">Monitor and control the background test scheduler</p>
        </Col>
    </Row>

    {#if loading}
        <div class="d-flex align-items-center gap-2">
            <Spinner size="sm" />
            <span>Loading scheduler status...</span>
        </div>
    {:else if error}
        <Card color="danger" body>
            <Icon name="exclamation-triangle-fill"></Icon>
            Error loading scheduler status: {error}
            <Button class="ms-3" size="sm" color="light" onclick={fetchStatus}>Retry</Button>
        </Card>
    {:else if status}
        <!-- Status Card -->
        <Card class="mb-4">
            <CardHeader>
                <div class="d-flex justify-content-between align-items-center">
                    <span><Icon name="info-circle-fill"></Icon> Scheduler Status</span>
                    <Button size="sm" color="secondary" outline onclick={fetchStatus}>
                        <Icon name="arrow-clockwise"></Icon> Refresh
                    </Button>
                </div>
            </CardHeader>
            <CardBody>
                <Row class="g-3 mb-3">
                    <Col sm={6} md={4}>
                        <div class="text-muted small">Config Enabled</div>
                        {#if status.config_enabled}
                            <Badge color="success">Yes</Badge>
                        {:else}
                            <Badge color="danger">No</Badge>
                        {/if}
                    </Col>
                    <Col sm={6} md={4}>
                        <div class="text-muted small">Runtime Enabled</div>
                        {#if status.runtime_enabled}
                            <Badge color="success">Yes</Badge>
                        {:else}
                            <Badge color="warning">Disabled</Badge>
                        {/if}
                    </Col>
                    <Col sm={6} md={4}>
                        <div class="text-muted small">Running</div>
                        {#if status.running}
                            <Badge color="success"><Icon name="play-fill"></Icon> Running</Badge>
                        {:else}
                            <Badge color="secondary"><Icon name="stop-fill"></Icon> Stopped</Badge>
                        {/if}
                    </Col>
                    <Col sm={6} md={4}>
                        <div class="text-muted small">Workers</div>
                        <strong>{status.worker_count}</strong>
                    </Col>
                    <Col sm={6} md={4}>
                        <div class="text-muted small">Queue Size</div>
                        <strong>{status.queue_size}</strong>
                    </Col>
                    <Col sm={6} md={4}>
                        <div class="text-muted small">Active Executions</div>
                        <strong>{status.active_count}</strong>
                    </Col>
                </Row>

                {#if status.config_enabled}
                    <div class="d-flex gap-2">
                        {#if status.runtime_enabled}
                            <Button
                                color="warning"
                                disabled={actionInProgress}
                                onclick={() => setEnabled(false)}
                            >
                                {#if actionInProgress}<Spinner size="sm" />{:else}<Icon
                                        name="pause-fill"
                                    ></Icon>{/if}
                                Disable Scheduler
                            </Button>
                        {:else}
                            <Button
                                color="success"
                                disabled={actionInProgress}
                                onclick={() => setEnabled(true)}
                            >
                                {#if actionInProgress}<Spinner size="sm" />{:else}<Icon
                                        name="play-fill"
                                    ></Icon>{/if}
                                Enable Scheduler
                            </Button>
                        {/if}
                        <Button
                            color="secondary"
                            outline
                            disabled={rescheduleInProgress}
                            onclick={rescheduleUpcoming}
                        >
                            {#if rescheduleInProgress}<Spinner size="sm" />{:else}<Icon
                                    name="shuffle"
                                ></Icon>{/if}
                            Spread Upcoming Checks
                        </Button>
                    </div>
                {:else}
                    <p class="text-muted mb-0">
                        <Icon name="lock-fill"></Icon>
                        The scheduler is disabled in the server configuration and cannot be enabled at
                        runtime.
                    </p>
                {/if}
            </CardBody>
        </Card>

        <!-- Upcoming Scheduled Checks -->
        <Card>
            <CardHeader>
                <Icon name="calendar-event-fill"></Icon>
                Upcoming Scheduled Checks
                {#if status.next_schedules}
                    <Badge color="secondary" class="ms-2">{status.next_schedules.length}</Badge>
                {/if}
            </CardHeader>
            <CardBody class="p-0">
                <div class="table-responsive">
                    <Table hover class="mb-0">
                        <thead>
                            <tr>
                                <th>Plugin</th>
                                <th>Target Type</th>
                                <th>Target ID</th>
                                <th>Interval</th>
                                <th>Last Run</th>
                                <th>Next Run</th>
                            </tr>
                        </thead>
                        <tbody>
                            {#if !status.next_schedules || status.next_schedules.length === 0}
                                <tr>
                                    <td colspan="6" class="text-center text-muted py-3">
                                        No scheduled checks
                                    </td>
                                </tr>
                            {:else}
                                {#each status.next_schedules as schedule}
                                    <tr>
                                        <td><strong>{schedule.checker_name}</strong></td>
                                        <td
                                            ><Badge color="info"
                                                >{targetTypeName(schedule.target_type)}</Badge
                                            ></td
                                        >
                                        <td><code class="small">{schedule.target_id}</code></td>
                                        <td>{formatDuration(schedule.interval)}</td>
                                        <td>
                                            {#if schedule.last_run}
                                                {new Date(schedule.last_run).toLocaleString()}
                                            {:else}
                                                <span class="text-muted">Never</span>
                                            {/if}
                                        </td>
                                        <td>
                                            {#if new Date(schedule.next_run) < new Date()}
                                                <span class="text-danger">
                                                    <Icon name="exclamation-circle-fill"></Icon>
                                                    {new Date(schedule.next_run).toLocaleString()}
                                                </span>
                                            {:else}
                                                {new Date(schedule.next_run).toLocaleString()}
                                            {/if}
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
