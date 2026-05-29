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
        Badge,
        Button,
        Container,
        Icon,
        Input,
        InputGroup,
        ListGroup,
        ListGroupItem,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import {
        addAvailabilityWatch,
        deleteAvailabilityWatch,
        getAvailabilityWatchStatus,
        triggerAvailabilityWatchCheck,
        type AvailabilityWatchStatus,
    } from "$lib/api/availability";
    import PageTitle from "$lib/components/PageTitle.svelte";
    import { availabilityWatches, refreshAvailabilityWatches } from "$lib/stores/availability";
    import { toasts } from "$lib/stores/toasts";
    import { t } from "$lib/translations";

    let newDomain = $state("");
    let adding = $state(false);

    // Latest availability status per watch id, fetched lazily after listing.
    let statuses = $state<Record<string, AvailabilityWatchStatus>>({});
    // Watches with an in-flight manual recheck.
    let rechecking = $state<Record<string, boolean>>({});

    const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

    // Set when the component is unmounted, so in-flight recheck polling loops
    // stop making requests instead of running until their deadline.
    let destroyed = false;
    onDestroy(() => {
        destroyed = true;
    });

    async function loadStatus(id: string) {
        try {
            statuses[id] = await getAvailabilityWatchStatus(id);
        } catch {
            // Leave the status undefined; the row simply shows no state.
        }
    }

    async function loadAllStatuses() {
        const watches = $availabilityWatches;
        if (!watches) return;
        await Promise.all(watches.map((w) => loadStatus(w.id)));
    }

    onMount(async () => {
        await refreshAvailabilityWatches();
        await loadAllStatuses();
    });

    async function onAdd() {
        const domain = newDomain.trim();
        if (!domain) return;

        adding = true;
        try {
            const watch = await addAvailabilityWatch({ domain });
            toasts.addToast({
                title: $t("availability.add"),
                message: $t("availability.added-success", { domain }),
                type: "success",
                timeout: 5000,
            });
            newDomain = "";
            await refreshAvailabilityWatches();
            await loadAllStatuses();

            // Kick off an immediate check so the new watch does not linger in
            // the "never checked" state. Fire and forget: the row shows its own
            // spinner while the recheck polls in the background.
            onRecheck(watch.id);
        } catch (err) {
            toasts.addErrorToast({
                title: $t("errors.error"),
                message: err instanceof Error ? err.message : String(err),
            });
        } finally {
            adding = false;
        }
    }

    async function onRecheck(id: string) {
        rechecking[id] = true;
        const previousChecked = statuses[id]?.last_checked;
        try {
            await triggerAvailabilityWatchCheck(id);

            // The check runs asynchronously on the server. Poll the status until
            // a new finished result appears (its last_checked changes) or we
            // give up after a reasonable delay.
            const deadline = Date.now() + 60000;
            for (;;) {
                await sleep(1500);
                if (destroyed) break;
                const status = await getAvailabilityWatchStatus(id);
                if (destroyed) break;
                statuses[id] = status;
                if (!status.checking && status.last_checked !== previousChecked) break;
                if (Date.now() > deadline) break;
            }
        } catch (err) {
            toasts.addErrorToast({
                title: $t("errors.error"),
                message: err instanceof Error ? err.message : String(err),
            });
        } finally {
            rechecking[id] = false;
        }
    }

    async function onDelete(id: string, domain: string) {
        if (!confirm($t("availability.delete-confirm", { domain }))) return;

        try {
            await deleteAvailabilityWatch(id);
            toasts.addToast({
                title: $t("availability.title"),
                message: $t("availability.deleted-success", { domain }),
                type: "success",
                timeout: 5000,
            });
            await refreshAvailabilityWatches();
        } catch (err) {
            toasts.addErrorToast({
                title: $t("errors.error"),
                message: err instanceof Error ? err.message : String(err),
            });
        }
    }
</script>

<svelte:head>
    <title>{$t("availability.title")} - happyDomain</title>
</svelte:head>

<Container class="flex-fill my-5">
    <PageTitle title={$t("availability.title")} subtitle={$t("availability.description")} />

    <form
        class="mb-4 mt-3"
        onsubmit={(e) => {
            e.preventDefault();
            onAdd();
        }}
    >
        <InputGroup>
            <Input
                type="text"
                placeholder={$t("availability.domain-placeholder")}
                bind:value={newDomain}
                disabled={adding}
            />
            <Button type="submit" color="primary" disabled={adding || !newDomain.trim()}>
                {#if adding}
                    <Spinner size="sm" />
                {:else}
                    <Icon name="plus" />
                {/if}
                {$t("availability.add")}
            </Button>
        </InputGroup>
    </form>

    {#if $availabilityWatches === undefined}
        <div class="text-center my-5">
            <Spinner />
        </div>
    {:else if $availabilityWatches.length === 0}
        <p class="text-muted text-center my-5">{$t("availability.empty")}</p>
    {:else}
        <ListGroup>
            {#each $availabilityWatches as watch (watch.id)}
                <ListGroupItem class="d-flex justify-content-between align-items-center">
                    {@const status = statuses[watch.id]}
                    <div>
                        <span class="font-monospace fw-bold">{watch.domain.replace(/\.$/, "")}</span>
                        <div class="small mt-1">
                            {#if rechecking[watch.id] || status?.checking}
                                <span class="text-muted">
                                    <Spinner size="sm" />
                                    {$t("availability.checking")}
                                </span>
                            {:else if status?.available === true}
                                <Badge color="success">{$t("availability.status-available")}</Badge>
                            {:else if status?.available === false}
                                <Badge color="secondary">{$t("availability.status-registered")}</Badge>
                            {:else if status?.error}
                                <span class="text-danger">{status.error}</span>
                            {:else}
                                <span class="text-muted">{$t("availability.never-checked")}</span>
                            {/if}
                            {#if status?.last_checked}
                                <small class="text-muted ms-2">
                                    {$t("availability.last-checked")}
                                    {new Date(status.last_checked).toLocaleString()}
                                </small>
                            {/if}
                        </div>
                    </div>
                    <div class="d-flex gap-2">
                        <Button
                            type="button"
                            color="secondary"
                            outline
                            size="sm"
                            disabled={rechecking[watch.id]}
                            onclick={() => onRecheck(watch.id)}
                        >
                            {#if rechecking[watch.id]}
                                <Spinner size="sm" />
                            {:else}
                                <Icon name="arrow-clockwise" />
                            {/if}
                            {$t("availability.check-now")}
                        </Button>
                        <Button
                            type="button"
                            color="danger"
                            outline
                            size="sm"
                            onclick={() => onDelete(watch.id, watch.domain)}
                        >
                            <Icon name="trash" />
                        </Button>
                    </div>
                </ListGroupItem>
            {/each}
        </ListGroup>
    {/if}
</Container>
