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
    import { resolve } from "$app/paths";
    import { Badge, Button, Card, CardBody, Input, Spinner } from "@sveltestrap/sveltestrap";

    import { checkDomainWatch, deleteDomainWatch, updateDomainWatch } from "$lib/api/watchlist";
    import DomainStatusBadges from "$lib/components/DomainStatusBadges.svelte";
    import LifecycleStepper from "$lib/components/watchlist/LifecycleStepper.svelte";
    import type { DomainWatch } from "$lib/model/watchlist";
    import { removeWatch, upsertWatch } from "$lib/stores/watchlist";
    import { toasts } from "$lib/stores/toasts";
    import { locale, t } from "$lib/translations";
    import { formatRelativeTime } from "$lib/utils/datetime";
    import { formatDate } from "$lib/utils/domaininfo";
    import {
        AUTO_RENEW_STYLE,
        isAutoRenewPending,
        nameOf,
        PHASE_STYLE,
        watchHeadline,
    } from "$lib/utils/watchlist";

    interface Props {
        watch: DomainWatch;
        // Draw attention to a watch just added.
        highlight?: boolean;
    }

    let { watch, highlight = false }: Props = $props();

    let phase = $derived(watch.phase ?? "unknown");
    let autoRenew = $derived(
        (phase === "registered" || phase === "expiring") && isAutoRenewPending(watch),
    );
    let style = $derived(autoRenew ? AUTO_RENEW_STYLE : PHASE_STYLE[phase]);
    let name = $derived(nameOf(watch));
    let headline = $derived($t(...watchHeadline(watch)));

    let checking = $state(false);
    let removing = $state(false);
    let editingNote = $state(false);
    let note = $state("");
    let savingNote = $state(false);
    let showHistory = $state(false);

    async function checkNow() {
        checking = true;
        try {
            upsertWatch(await checkDomainWatch(watch.id!));
        } catch (err) {
            toasts.addErrorToast({
                title: $t("watchlist.errors.check"),
                message: String(err),
                timeout: 8000,
            });
        } finally {
            checking = false;
        }
    }

    async function remove() {
        if (!confirm($t("watchlist.confirm-remove", { domain: name }))) return;
        removing = true;
        try {
            await deleteDomainWatch(watch.id!);
            removeWatch(watch.id!);
        } catch (err) {
            toasts.addErrorToast({
                title: $t("watchlist.errors.remove"),
                message: String(err),
                timeout: 8000,
            });
            removing = false;
        }
    }

    function editNote() {
        note = watch.note ?? "";
        editingNote = true;
    }

    async function saveNote(e: SubmitEvent) {
        e.preventDefault();
        savingNote = true;
        try {
            upsertWatch(await updateDomainWatch(watch.id!, note.trim()));
            editingNote = false;
        } catch (err) {
            toasts.addErrorToast({
                title: $t("watchlist.errors.note"),
                message: String(err),
                timeout: 8000,
            });
        } finally {
            savingNote = false;
        }
    }
</script>

<Card class="h-100 watch-card border-{style.color} {highlight ? 'highlight' : ''}">
    <CardBody class="d-flex flex-column gap-3">
        <div class="d-flex align-items-start justify-content-between gap-2">
            <div class="min-w-0">
                <h3 class="h5 font-monospace fw-bold mb-1 text-break">{name}</h3>
                {#if watch.displayName && watch.domain && watch.displayName !== watch.domain}
                    <div class="small text-muted font-monospace text-break">{watch.domain}</div>
                {/if}
            </div>
            <Badge
                color={style.color}
                class="flex-shrink-0 {phase === 'unknown' ? 'text-dark border' : ''}"
            >
                <i class="bi bi-{style.icon} me-1"></i>
                {autoRenew ? $t("watchlist.auto-renew") : $t(`watchlist.phases.${phase}`)}
            </Badge>
        </div>

        <div>
            <p
                class="headline fw-semibold mb-1 text-{phase === 'unknown' || phase === 'registered'
                    ? 'body'
                    : style.color}"
            >
                {headline}
            </p>
            <p class="small text-muted mb-0">
                {#if phase === "available"}
                    {$t("watchlist.hints.available")}
                {:else if phase === "pending_delete" || phase === "redemption"}
                    {#if watch.estimatedRelease}
                        {$t("watchlist.hints.release-by", {
                            date: formatDate(watch.estimatedRelease),
                        })}
                    {/if}
                    {$t(`watchlist.hints.${phase}`)}
                {:else if phase === "grace"}
                    {$t("watchlist.hints.grace")}
                {:else if (phase === "expiring" || phase === "registered") && watch.expiration}
                    {$t("watchlist.hints.expires-on", { date: formatDate(watch.expiration) })}
                    {#if autoRenew}
                        {$t("watchlist.hints.auto-renew")}
                    {/if}
                {/if}
            </p>
        </div>

        <LifecycleStepper phase={watch.phase} />

        {#if watch.status && watch.status.length > 0}
            <DomainStatusBadges status={watch.status} describeSingle={false} />
        {/if}

        {#if editingNote}
            <form class="d-flex gap-2" onsubmit={saveNote}>
                <Input
                    bsSize="sm"
                    bind:value={note}
                    maxlength={1000}
                    placeholder={$t("watchlist.note-placeholder")}
                    autofocus
                />
                <Button
                    size="sm"
                    color="primary"
                    type="submit"
                    title={$t("common.save")}
                    disabled={savingNote}
                >
                    {#if savingNote}<Spinner size="sm" />{:else}<i class="bi bi-check-lg"></i>{/if}
                </Button>
                <Button
                    size="sm"
                    color="secondary"
                    outline
                    type="button"
                    title={$t("common.cancel")}
                    onclick={() => (editingNote = false)}
                >
                    <i class="bi bi-x-lg"></i>
                </Button>
            </form>
        {:else if watch.note}
            <button
                type="button"
                class="note btn btn-link p-0 text-start text-body small"
                onclick={editNote}
                title={$t("watchlist.edit-note")}
            >
                <i class="bi bi-sticky me-1 text-muted"></i><em>{watch.note}</em>
            </button>
        {/if}

        {#if watch.lastError}
            <p class="small text-warning-emphasis mb-0">
                <i class="bi bi-exclamation-triangle me-1"></i>
                {$t("watchlist.last-error", { error: watch.lastError })}
            </p>
        {/if}

        {#if showHistory && watch.history && watch.history.length > 0}
            <ul class="small list-unstyled mb-0 border-start ps-3">
                {#each [...watch.history].reverse() as ev, i (i)}
                    <li>
                        <span class="text-muted">{formatDate(ev.at)}</span>
                        — {$t(`watchlist.phases.${ev.from ?? "unknown"}`)}
                        <i class="bi bi-arrow-right"></i>
                        {$t(`watchlist.phases.${ev.to ?? "unknown"}`)}
                    </li>
                {/each}
            </ul>
        {/if}

        <div
            class="mt-auto pt-2 border-top d-flex flex-wrap align-items-center justify-content-between gap-2"
        >
            <div class="small text-muted">
                {#if watch.registrar && watch.registrar !== "Unknown"}
                    <span class="me-2"><i class="bi bi-building me-1"></i>{watch.registrar}</span>
                {/if}
                <span
                    title={$t("watchlist.next-check", {
                        when: formatRelativeTime(watch.nextCheck, $locale),
                    })}
                >
                    <i class="bi bi-clock-history me-1"></i>
                    {#if watch.lastCheck}
                        {$t("watchlist.last-check", {
                            when: formatRelativeTime(watch.lastCheck, $locale),
                        })}
                    {:else}
                        {$t("watchlist.never-checked")}
                    {/if}
                </span>
            </div>
            <div class="d-flex gap-1">
                {#if watch.history && watch.history.length > 0}
                    <Button
                        size="sm"
                        color="secondary"
                        outline
                        active={showHistory}
                        title={$t("watchlist.history")}
                        onclick={() => (showHistory = !showHistory)}
                    >
                        <i class="bi bi-list-ul"></i>
                    </Button>
                {/if}
                {#if !editingNote}
                    <Button
                        size="sm"
                        color="secondary"
                        outline
                        title={$t("watchlist.edit-note")}
                        onclick={editNote}
                    >
                        <i class="bi bi-pencil"></i>
                    </Button>
                {/if}
                <Button
                    size="sm"
                    color="secondary"
                    outline
                    href={resolve("/whois/[[domain]]", {
                        domain: encodeURIComponent(watch.domain ?? ""),
                    })}
                    title={$t("watchlist.details")}
                >
                    <i class="bi bi-info-circle"></i>
                </Button>
                <Button
                    size="sm"
                    color="primary"
                    outline
                    title={$t("watchlist.check-now")}
                    disabled={checking}
                    onclick={checkNow}
                >
                    {#if checking}<Spinner size="sm" />{:else}<i class="bi bi-arrow-clockwise"
                        ></i>{/if}
                </Button>
                <Button
                    size="sm"
                    color="danger"
                    outline
                    title={$t("watchlist.remove")}
                    disabled={removing}
                    onclick={remove}
                >
                    <i class="bi bi-trash"></i>
                </Button>
            </div>
        </div>
    </CardBody>
</Card>

<style>
    :global(.watch-card) {
        border-width: 1px;
        transition: box-shadow 0.3s;
    }

    :global(.watch-card.highlight) {
        animation: watch-added 2s ease-out;
    }

    @keyframes watch-added {
        from {
            box-shadow: 0 0 0 0.4rem rgba(var(--bs-primary-rgb), 0.35);
        }
        to {
            box-shadow: 0 0 0 0 rgba(var(--bs-primary-rgb), 0);
        }
    }

    .headline {
        font-size: 1.25rem;
    }

    .min-w-0 {
        min-width: 0;
    }

    .note {
        text-decoration: none;
    }
</style>
