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
    import { onMount } from "svelte";
    import { flip } from "svelte/animate";
    import { fade } from "svelte/transition";

    import {
        Button,
        Card,
        CardBody,
        Col,
        Container,
        Dropdown,
        DropdownItem,
        DropdownMenu,
        DropdownToggle,
        Input,
        InputGroup,
        InputGroupText,
        Row,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { addDomainWatch } from "$lib/api/watchlist";
    import PageTitle from "$lib/components/PageTitle.svelte";
    import { validateDomain } from "$lib/dns";
    import WatchCard from "$lib/components/watchlist/WatchCard.svelte";
    import { refreshWatchlist, upsertWatch, watchlist } from "$lib/stores/watchlist";
    import { toasts } from "$lib/stores/toasts";
    import { t } from "$lib/translations";
    import { WATCH_SORTS, watchComparator, type WatchSort } from "$lib/utils/watchlist";

    let loadError = $state<string | null>(null);

    let domain = $state("");
    let note = $state("");
    let showNote = $state(false);
    let adding = $state(false);
    let addError = $state<string | null>(null);
    let lastAdded = $state<string | null>(null);

    // A bare label (e.g. "example") is a TLD, not a registrable domain: require
    // at least one dot between two labels, ignoring the optional trailing one.
    let domainState = $derived.by(() => {
        const dn = domain.trim();
        const state = validateDomain(dn, "", false);
        if (state && !dn.replace(/\.$/, "").includes(".")) return false;
        return state;
    });

    let sortBy = $state<WatchSort>("urgency");
    let sorted = $derived($watchlist ? [...$watchlist].sort(watchComparator(sortBy)) : []);
    let needle = $derived(domain.trim().toLowerCase());
    let filtered = $derived(
        needle
            ? sorted.filter((w) =>
                  [w.domain, w.displayName].some((n) => n?.toLowerCase().includes(needle)),
              )
            : sorted,
    );

    onMount(() => {
        // The store may already hold the list, loaded by another page: it is
        // shown at once, while refreshed.
        refreshWatchlist().then(
            () => (loadError = null),
            (err) => {
                loadError = String(err);
                // The list shown may be outdated: tell it.
                if ($watchlist !== undefined) {
                    toasts.addErrorToast({
                        title: $t("watchlist.errors.load"),
                        message: String(err),
                        timeout: 8000,
                    });
                }
            },
        );
    });

    async function add(e: SubmitEvent) {
        e.preventDefault();
        if (!domain.trim() || !domainState) return;

        adding = true;
        addError = null;
        try {
            const w = await addDomainWatch({ domain: domain.trim(), note: note.trim() });
            if ($watchlist === undefined) await refreshWatchlist();
            else upsertWatch(w);
            lastAdded = w.id ?? null;
            domain = "";
            note = "";
            showNote = false;
        } catch (err) {
            addError = err instanceof Error ? err.message : String(err);
        } finally {
            adding = false;
        }
    }

    function retry() {
        loadError = null;
        refreshWatchlist().catch((err) => {
            loadError = String(err);
            toasts.addErrorToast({
                title: $t("watchlist.errors.load"),
                message: String(err),
                timeout: 8000,
            });
        });
    }
</script>

<svelte:head>
    <title>{$t("watchlist.title")} - happyDomain</title>
</svelte:head>

<Container class="flex-fill my-5">
    <PageTitle title={$t("watchlist.title")} subtitle={$t("watchlist.description")}>
        <p class="small text-muted mb-0">
            <i class="bi bi-bell me-1"></i>
            {$t("watchlist.notifications-hint")}
            <a href={resolve("/me/notifications")}>{$t("watchlist.notifications-link")}</a>
        </p>
    </PageTitle>

    <Row class="mb-4">
        <Col lg={{ offset: 2, size: 8 }}>
            <form onsubmit={add}>
                <InputGroup size="lg">
                    <InputGroupText>
                        <i class="bi bi-search"></i>
                    </InputGroupText>
                    <Input
                        type="text"
                        class="font-monospace"
                        placeholder="example.com"
                        aria-label={$t("watchlist.domain-label")}
                        autofocus
                        autocapitalize="off"
                        autocomplete="off"
                        spellcheck="false"
                        bind:value={domain}
                        disabled={adding}
                        invalid={addError !== null || domainState === false}
                        valid={addError === null && domainState === true}
                        oninput={() => (addError = null)}
                    />
                    <Button type="submit" color="primary" disabled={adding || !domainState}>
                        {#if adding}
                            <Spinner size="sm" class="me-1" />
                            {$t("watchlist.adding")}
                        {:else}
                            <i class="bi bi-plus-lg me-1"></i>
                            {$t("watchlist.add")}
                        {/if}
                    </Button>
                </InputGroup>
                {#if addError}
                    <div class="small text-danger mt-1">{addError}</div>
                {/if}
                {#if showNote}
                    <Input
                        type="text"
                        bsSize="sm"
                        class="mt-2"
                        maxlength={1000}
                        placeholder={$t("watchlist.note-placeholder")}
                        bind:value={note}
                        disabled={adding}
                    />
                {:else}
                    <button
                        type="button"
                        class="btn btn-link btn-sm px-0"
                        onclick={() => (showNote = true)}
                    >
                        <i class="bi bi-sticky me-1"></i>{$t("watchlist.add-note")}
                    </button>
                {/if}
            </form>
        </Col>
    </Row>

    {#if loadError && $watchlist === undefined}
        <Card body color="danger" outline>
            <p class="mb-2">{$t("watchlist.errors.load")}: {loadError}</p>
            <Button size="sm" color="danger" outline onclick={retry}>{$t("watchlist.retry")}</Button
            >
        </Card>
    {:else if $watchlist === undefined}
        <Card body>
            <p class="text-center mb-0">
                <Spinner size="sm" class="me-2" />
                {$t("watchlist.loading")}
            </p>
        </Card>
    {:else if sorted.length === 0}
        <div in:fade>
            <Card class="empty-state border-0 bg-body-tertiary">
                <CardBody class="p-4 p-md-5">
                    <h2 class="h4 mb-3">
                        <i class="bi bi-clock-history text-primary me-2"></i>{$t(
                            "watchlist.empty.title",
                        )}
                    </h2>
                    <p class="text-muted">{$t("watchlist.empty.intro")}</p>
                    <Row class="g-4 mt-1">
                        {#each ["add", "follow", "notify"] as step, i (step)}
                            <Col md={4}>
                                <div class="d-flex gap-3">
                                    <span class="step-number">{i + 1}</span>
                                    <div>
                                        <h3 class="h6 fw-bold mb-1">
                                            {$t(`watchlist.empty.${step}.title`)}
                                        </h3>
                                        <p class="small text-muted mb-0">
                                            {$t(`watchlist.empty.${step}.text`)}
                                        </p>
                                    </div>
                                </div>
                            </Col>
                        {/each}
                    </Row>
                </CardBody>
            </Card>
        </div>
    {:else if filtered.length === 0}
        <p class="text-center text-muted">{$t("watchlist.no-match", { filter: domain.trim() })}</p>
    {:else}
        <div class="d-flex justify-content-end mb-3">
            <Dropdown>
                <DropdownToggle color="secondary" outline size="sm" caret>
                    <i class="bi bi-{WATCH_SORTS.find((s) => s.key === sortBy)?.icon} me-1"></i>
                    {$t("watchlist.sort.label")}: {$t(`watchlist.sort.${sortBy}`)}
                </DropdownToggle>
                <DropdownMenu end>
                    {#each WATCH_SORTS as s (s.key)}
                        <DropdownItem active={sortBy === s.key} onclick={() => (sortBy = s.key)}>
                            <i class="bi bi-{s.icon} me-2"></i>{$t(`watchlist.sort.${s.key}`)}
                        </DropdownItem>
                    {/each}
                </DropdownMenu>
            </Dropdown>
        </div>
        <Row class="g-4" cols={{ xs: 1, lg: 2 }}>
            {#each filtered as w (w.id)}
                <div class="col" animate:flip={{ duration: 300 }}>
                    <WatchCard watch={w} highlight={w.id === lastAdded} />
                </div>
            {/each}
        </Row>
    {/if}
</Container>

<style>
    .step-number {
        flex-shrink: 0;
        display: flex;
        align-items: center;
        justify-content: center;
        width: 2rem;
        height: 2rem;
        border-radius: 50%;
        background: var(--bs-primary);
        color: #fff;
        font-weight: 700;
    }
</style>
