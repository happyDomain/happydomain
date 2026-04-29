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

    import { Badge, Button, Spinner, Table } from "@sveltestrap/sveltestrap";

    import { listHistory, type NotificationRecord } from "$lib/api/notifications";
    import { toasts } from "$lib/stores/toasts";
    import { t } from "$lib/translations";
    import {
        getStatusColor,
        getStatusI18nKey,
    } from "$lib/utils/checkers";

    let records: NotificationRecord[] = $state([]);
    let loading: boolean = $state(true);
    let limit: number = $state(50);
    let loadingMore: boolean = $state(false);

    async function refresh(newLimit?: number) {
        const lim = newLimit ?? limit;
        const isFirst = !newLimit;
        if (isFirst) loading = true;
        else loadingMore = true;
        try {
            records = await listHistory(lim);
            limit = lim;
        } catch (e) {
            toasts.addErrorToast({
                title: $t("settings.notifications.history.loadError"),
                message: String(e),
                timeout: 8000,
            });
        } finally {
            loading = false;
            loadingMore = false;
        }
    }

    onMount(() => {
        refresh();
    });

    function formatTarget(t: NotificationRecord["target"]): string {
        if (!t) return "—";
        const parts: string[] = [];
        if (t.domainId) parts.push(`d:${t.domainId.slice(0, 8)}`);
        if (t.serviceId) parts.push(`s:${t.serviceId.slice(0, 8)}`);
        if (t.serviceType) parts.push(t.serviceType);
        return parts.join(" / ") || "—";
    }

    function formatDate(d: Date | string | undefined): string {
        if (!d) return "—";
        try {
            return d instanceof Date ? d.toLocaleString() : new Date(d).toLocaleString();
        } catch {
            return String(d);
        }
    }
</script>

<div class="d-flex justify-content-between align-items-center mb-3">
    <p class="mb-0 text-muted">
        {$t("settings.notifications.history.description")}
    </p>
    <Button
        color="secondary"
        outline
        size="sm"
        on:click={() => refresh()}
        disabled={loading || loadingMore}
        title={$t("settings.notifications.history.refresh")}
    >
        <i class="bi bi-arrow-clockwise"></i>
    </Button>
</div>

{#if loading}
    <div class="d-flex justify-content-center py-3">
        <Spinner color="primary" />
    </div>
{:else if records.length === 0}
    <div class="alert alert-secondary">
        {$t("settings.notifications.history.empty")}
    </div>
{:else}
    <div class="table-responsive">
        <Table size="sm" hover>
            <thead>
                <tr>
                    <th>{$t("settings.notifications.history.sentAt")}</th>
                    <th>{$t("settings.notifications.history.channel")}</th>
                    <th>{$t("settings.notifications.history.checker")}</th>
                    <th>{$t("settings.notifications.history.target")}</th>
                    <th>{$t("settings.notifications.history.transition")}</th>
                    <th>{$t("settings.notifications.history.result")}</th>
                </tr>
            </thead>
            <tbody>
                {#each records as r (r.id)}
                    <tr>
                        <td>{formatDate(r.sentAt)}</td>
                        <td>
                            <Badge color="info">{r.channelType}</Badge>
                        </td>
                        <td><code>{r.checkerId}</code></td>
                        <td><small>{formatTarget(r.target)}</small></td>
                        <td>
                            <Badge color={getStatusColor(r.oldStatus)}>
                                {$t(getStatusI18nKey(r.oldStatus))}
                            </Badge>
                            →
                            <Badge color={getStatusColor(r.newStatus)}>
                                {$t(getStatusI18nKey(r.newStatus))}
                            </Badge>
                        </td>
                        <td>
                            {#if r.success}
                                <Badge color="success">
                                    <i class="bi bi-check-lg"></i>
                                    {$t("settings.notifications.history.success")}
                                </Badge>
                            {:else}
                                <Badge color="danger" title={r.error}>
                                    <i class="bi bi-x-lg"></i>
                                    {$t("settings.notifications.history.failure")}
                                </Badge>
                                {#if r.error}
                                    <div class="small text-muted">{r.error}</div>
                                {/if}
                            {/if}
                        </td>
                    </tr>
                {/each}
            </tbody>
        </Table>
    </div>

    {#if records.length >= limit}
        <div class="d-flex justify-content-center">
            <Button
                color="secondary"
                outline
                size="sm"
                on:click={() => refresh(limit + 50)}
                disabled={loadingMore}
            >
                {#if loadingMore}<Spinner size="sm" class="me-2" />{/if}
                {$t("settings.notifications.history.loadMore")}
            </Button>
        </div>
    {/if}
{/if}
