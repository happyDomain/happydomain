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
        deleteChannel,
        listChannels,
        testChannel,
        type NotificationChannel,
    } from "$lib/api/notifications";
    import {
        notificationChannelTypes,
        refreshNotificationChannelTypes,
    } from "$lib/stores/notificationTypes";
    import { toasts } from "$lib/stores/toasts";
    import { t } from "$lib/translations";

    import ChannelEditor from "./ChannelEditor.svelte";

    let channels: NotificationChannel[] = $state([]);
    let loading: boolean = $state(true);
    let editorOpen: boolean = $state(false);
    let editing: NotificationChannel | null = $state(null);
    let busy: Record<string, "test" | "delete"> = $state({});

    async function refresh() {
        loading = true;
        try {
            channels = await listChannels();
        } catch (e) {
            toasts.addErrorToast({
                title: $t("settings.notifications.channels.loadError"),
                message: String(e),
                timeout: 8000,
            });
        } finally {
            loading = false;
        }
    }

    onMount(() => {
        refresh();
        refreshNotificationChannelTypes().catch((e) =>
            toasts.addErrorToast({
                title: $t("settings.notifications.channels.typesError"),
                message: String(e),
                timeout: 8000,
            }),
        );
    });

    function openCreate() {
        editing = null;
        editorOpen = true;
    }

    function openEdit(channel: NotificationChannel) {
        editing = channel;
        editorOpen = true;
    }

    async function onTest(channel: NotificationChannel) {
        if (!channel.id) return;
        busy = { ...busy, [channel.id]: "test" };
        try {
            await testChannel(channel.id);
            toasts.addToast({
                title: $t("settings.notifications.channels.testSent"),
                timeout: 5000,
                type: "success",
            });
        } catch (e) {
            toasts.addErrorToast({
                title: $t("settings.notifications.channels.testError"),
                message: String(e),
                timeout: 8000,
            });
        } finally {
            const next = { ...busy };
            delete next[channel.id];
            busy = next;
        }
    }

    async function onDelete(channel: NotificationChannel) {
        if (!channel.id) return;
        if (!window.confirm($t("settings.notifications.channels.confirmDelete"))) return;
        busy = { ...busy, [channel.id]: "delete" };
        try {
            await deleteChannel(channel.id);
            channels = channels.filter((c) => c.id !== channel.id);
            toasts.addToast({
                title: $t("settings.notifications.channels.deleted"),
                timeout: 4000,
                type: "success",
            });
        } catch (e) {
            toasts.addErrorToast({
                title: $t("settings.notifications.channels.deleteError"),
                message: String(e),
                timeout: 8000,
            });
        } finally {
            const next = { ...busy };
            delete next[channel.id];
            busy = next;
        }
    }

    function onSaved() {
        editorOpen = false;
        refresh();
    }
</script>

<div class="d-flex justify-content-between align-items-center mb-3">
    <p class="mb-0 text-muted">
        {$t("settings.notifications.channels.description")}
    </p>
    <Button color="primary" size="sm" on:click={openCreate} disabled={!$notificationChannelTypes}>
        <i class="bi bi-plus-lg"></i>
        {$t("settings.notifications.channels.add")}
    </Button>
</div>

{#if loading}
    <div class="d-flex justify-content-center py-3">
        <Spinner color="primary" />
    </div>
{:else if channels.length === 0}
    <div class="alert alert-secondary">
        {$t("settings.notifications.channels.empty")}
    </div>
{:else}
    <ul class="list-group">
        {#each channels as channel (channel.id)}
            <li class="list-group-item d-flex flex-column flex-md-row gap-2 align-items-md-center">
                <div class="flex-grow-1">
                    <div class="d-flex align-items-center gap-2">
                        <strong>{channel.name || $t("settings.notifications.channels.unnamed")}</strong>
                        <Badge color="info">{channel.type}</Badge>
                        {#if !channel.enabled}
                            <Badge color="secondary">
                                {$t("settings.notifications.channels.disabled")}
                            </Badge>
                        {/if}
                    </div>
                </div>
                <div class="d-flex gap-2">
                    <Button
                        size="sm"
                        color="secondary"
                        outline
                        on:click={() => onTest(channel)}
                        disabled={busy[channel.id ?? ""] === "test" || !channel.enabled}
                        title={$t("settings.notifications.channels.test")}
                    >
                        {#if busy[channel.id ?? ""] === "test"}
                            <Spinner size="sm" />
                        {:else}
                            <i class="bi bi-send"></i>
                        {/if}
                    </Button>
                    <Button
                        size="sm"
                        color="secondary"
                        outline
                        on:click={() => openEdit(channel)}
                        title={$t("settings.notifications.channels.edit")}
                    >
                        <i class="bi bi-pencil"></i>
                    </Button>
                    <Button
                        size="sm"
                        color="danger"
                        outline
                        on:click={() => onDelete(channel)}
                        disabled={busy[channel.id ?? ""] === "delete"}
                        title={$t("settings.notifications.channels.delete")}
                    >
                        {#if busy[channel.id ?? ""] === "delete"}
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

<ChannelEditor
    open={editorOpen}
    channelTypes={$notificationChannelTypes ?? []}
    channel={editing}
    onSave={onSaved}
    onClose={() => (editorOpen = false)}
/>
