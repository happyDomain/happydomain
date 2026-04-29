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

    import { Badge, Button, Spinner } from "@sveltestrap/sveltestrap";

    import {
        deletePreference,
        listChannels,
        listPreferences,
        type NotificationChannel,
        type NotificationPreference,
    } from "$lib/api/notifications";
    import { domains, refreshDomains } from "$lib/stores/domains";
    import { toasts } from "$lib/stores/toasts";
    import { t } from "$lib/translations";
    import { getStatusColor, getStatusI18nKey } from "$lib/utils/checkers";

    import PreferenceEditor from "./PreferenceEditor.svelte";

    let preferences: NotificationPreference[] = $state([]);
    let channels: NotificationChannel[] = $state([]);
    let loading: boolean = $state(true);
    let editorOpen: boolean = $state(false);
    let editing: NotificationPreference | null = $state(null);
    let deleting: Record<string, boolean> = $state({});

    async function refresh() {
        loading = true;
        try {
            const [p, c] = await Promise.all([listPreferences(), listChannels()]);
            preferences = p;
            channels = c;
        } catch (e) {
            toasts.addErrorToast({
                title: $t("settings.notifications.preferences.loadError"),
                message: String(e),
                timeout: 8000,
            });
        } finally {
            loading = false;
        }
    }

    onMount(() => {
        refresh();
        if (!$domains) refreshDomains().catch(() => {});
    });

    function openCreate() {
        editing = null;
        editorOpen = true;
    }

    function openEdit(pref: NotificationPreference) {
        editing = pref;
        editorOpen = true;
    }

    async function onDelete(pref: NotificationPreference) {
        if (!pref.id) return;
        if (!window.confirm($t("settings.notifications.preferences.confirmDelete"))) return;
        deleting = { ...deleting, [pref.id]: true };
        try {
            await deletePreference(pref.id);
            preferences = preferences.filter((p) => p.id !== pref.id);
            toasts.addToast({
                title: $t("settings.notifications.preferences.deleted"),
                timeout: 4000,
                type: "success",
            });
        } catch (e) {
            toasts.addErrorToast({
                title: $t("settings.notifications.preferences.deleteError"),
                message: String(e),
                timeout: 8000,
            });
        } finally {
            const next = { ...deleting };
            delete next[pref.id];
            deleting = next;
        }
    }

    function scopeLabel(p: NotificationPreference): string {
        if (p.serviceId) {
            const dom = $domains?.find((d) => d.id === p.domainId);
            const domLabel = dom ? dom.domain : p.domainId ?? "?";
            return $t("settings.notifications.preferences.scope.serviceLabel", {
                domain: domLabel,
                service: p.serviceId,
            });
        }
        if (p.domainId) {
            const dom = $domains?.find((d) => d.id === p.domainId);
            return dom ? dom.domain : p.domainId;
        }
        return $t("settings.notifications.preferences.scope.global");
    }

    function channelsLabel(p: NotificationPreference): string {
        if (!p.channelIds || p.channelIds.length === 0) {
            return $t("settings.notifications.preferences.allChannels");
        }
        return p.channelIds
            .map((id) => channels.find((c) => c.id === id)?.name || id)
            .join(", ");
    }

    function onSaved() {
        editorOpen = false;
        refresh();
    }

    let hasGlobalPreference = $derived(
        preferences.some((p) => !p.domainId && !p.serviceId && p.enabled),
    );
</script>

<div class="d-flex justify-content-between align-items-center mb-3">
    <p class="mb-0 text-muted">
        {$t("settings.notifications.preferences.description")}
    </p>
    <Button color="primary" size="sm" on:click={openCreate}>
        <i class="bi bi-plus-lg"></i>
        {$t("settings.notifications.preferences.add")}
    </Button>
</div>

{#if loading}
    <div class="d-flex justify-content-center py-3">
        <Spinner color="primary" />
    </div>
{:else}
    {#if !hasGlobalPreference}
        <div class="card mb-3 border-info">
            <div class="card-body py-2">
                <div class="d-flex align-items-center gap-2 mb-1">
                    <i class="bi bi-info-circle text-info"></i>
                    <strong>{$t("settings.notifications.preferences.defaults.title")}</strong>
                </div>
                <p class="mb-2 text-muted small">
                    {$t("settings.notifications.preferences.defaults.description")}
                </p>
                <ul class="list-unstyled mb-1 small">
                    <li>
                        <i class="bi bi-bell text-warning"></i>
                        {$t("settings.notifications.preferences.defaults.minStatus")}
                    </li>
                    <li>
                        <i class="bi bi-broadcast"></i>
                        {$t("settings.notifications.preferences.defaults.channels")}
                    </li>
                    <li>
                        <i class="bi bi-dash-circle text-secondary"></i>
                        {$t("settings.notifications.preferences.defaults.noRecovery")}
                    </li>
                </ul>
                <small class="text-muted fst-italic">
                    {$t("settings.notifications.preferences.defaults.override")}
                </small>
            </div>
        </div>
    {/if}
{#if preferences.length === 0}
    <div class="alert alert-secondary">
        {$t("settings.notifications.preferences.empty")}
    </div>
{:else}
    <ul class="list-group">
        {#each preferences as pref (pref.id)}
            <li class="list-group-item d-flex flex-column flex-md-row gap-2 align-items-md-center">
                <div class="flex-grow-1">
                    <div class="d-flex align-items-center gap-2 flex-wrap">
                        <strong>{scopeLabel(pref)}</strong>
                        {#if !pref.enabled}
                            <Badge color="secondary">
                                {$t("settings.notifications.preferences.disabled")}
                            </Badge>
                        {/if}
                        <Badge color={getStatusColor(pref.minStatus)}>
                            ≥ {$t(getStatusI18nKey(pref.minStatus))}
                        </Badge>
                        {#if pref.notifyRecovery}
                            <Badge color="success">
                                <i class="bi bi-arrow-counterclockwise"></i>
                                {$t("settings.notifications.preferences.recovery")}
                            </Badge>
                        {/if}
                        {#if pref.quietStart !== undefined && pref.quietEnd !== undefined && pref.quietStart !== null && pref.quietEnd !== null}
                            <Badge color="info">
                                <i class="bi bi-moon"></i>
                                {pref.quietStart}h–{pref.quietEnd}h UTC
                            </Badge>
                        {/if}
                    </div>
                    <small class="text-muted">{channelsLabel(pref)}</small>
                </div>
                <div class="d-flex gap-2">
                    <Button
                        size="sm"
                        color="secondary"
                        outline
                        on:click={() => openEdit(pref)}
                        title={$t("settings.notifications.preferences.edit")}
                    >
                        <i class="bi bi-pencil"></i>
                    </Button>
                    <Button
                        size="sm"
                        color="danger"
                        outline
                        on:click={() => onDelete(pref)}
                        disabled={deleting[pref.id ?? ""]}
                        title={$t("settings.notifications.preferences.delete")}
                    >
                        {#if deleting[pref.id ?? ""]}
                            <Spinner size="sm" />
                        {:else}
                            <i class="bi bi-trash"></i>
                        {/if}
                    </Button>
                </div>
            </li>
        {/each}
    </ul>
{/if}
{/if}

<PreferenceEditor
    open={editorOpen}
    preference={editing}
    {channels}
    onSave={onSaved}
    onClose={() => (editorOpen = false)}
/>
