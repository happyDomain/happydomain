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
    import { goto } from "$app/navigation";
    import { resolve } from "$app/paths";
    import { untrack } from "svelte";
    import { fly, fade } from "svelte/transition";
    import { cubicOut } from "svelte/easing";

    import { Spinner } from "@sveltestrap/sveltestrap";

    import {
        type DomainInfoState,
        createDomainInfoState,
        fetchDomainInfo,
    } from "$lib/components/DomainInfoFetcher.svelte";
    import DomainInfoDisplay from "$lib/components/DomainInfoDisplay.svelte";
    import DomainInfoLookupForm from "$lib/components/DomainInfoLookupForm.svelte";
    import { t } from "$lib/translations";
    import LookupLayout, { MORPH_DURATION } from "$lib/components/LookupLayout.svelte";
    import WatchCard from "$lib/components/watchlist/WatchCard.svelte";
    import { ensureWatchlist, watchlist } from "$lib/stores/watchlist";
    import { userSession } from "$lib/stores/usersession";
    import { findWatch, updatedWatch, watchOfDomainInfo } from "$lib/utils/watchlist";

    interface Props {
        data: { domain?: string };
    }

    let { data }: Props = $props();

    let domain = $derived(data.domain ?? "");
    let inputDomain = $state("");

    let lookup: DomainInfoState = $state(createDomainInfoState());

    $effect(() => {
        if ($userSession.id) {
            // Without the list, the page shows what the lookup tells.
            ensureWatchlist().catch(() => {});
        }
    });

    // The watch of the domain when it is in the watchlist, updated with what
    // the lookup tells, else what the lookup tells alone.
    let watch = $derived.by(() => {
        const lookedUp =
            lookup.info !== null || lookup.notFound
                ? watchOfDomainInfo(domain, lookup.info)
                : undefined;
        const saved = findWatch($watchlist, domain);
        if (!saved) return lookedUp;
        return lookedUp ? updatedWatch(saved, domain, lookedUp) : saved;
    });

    $effect(() => {
        if (domain) {
            untrack(() => {
                inputDomain = domain;
                fetchDomainInfo(domain, lookup);
            });
        }
    });

    function submit() {
        if (!inputDomain) return;

        if (inputDomain === domain) {
            fetchDomainInfo(inputDomain, lookup);
        } else {
            goto(resolve("/whois/[[domain]]", { domain: encodeURIComponent(inputDomain) }), {
                noScroll: true,
            });
        }
    }
</script>

<svelte:head>
    <title>
        {$t("domaininfo.page-title")}
        {domain ? domain : ""}
        - happyDomain
    </title>
</svelte:head>

<LookupLayout
    expanded={!!domain}
    title={$t("domaininfo.page-title")}
    {domain}
    subtitle={$t("domaininfo.page-description")}
    guide={{
        title: $t("domaininfo.guide.title"),
        intro: $t("domaininfo.guide.intro"),
        icon: "card-text",
        uses: [
            {
                icon: "calendar-event",
                title: $t("domaininfo.guide.expiration.title"),
                text: $t("domaininfo.guide.expiration.text"),
            },
            {
                icon: "building",
                title: $t("domaininfo.guide.registrar.title"),
                text: $t("domaininfo.guide.registrar.text"),
            },
            {
                icon: "diagram-3",
                title: $t("domaininfo.guide.delegation.title"),
                text: $t("domaininfo.guide.delegation.text"),
            },
        ],
    }}
>
    {#snippet search()}
        <DomainInfoLookupForm
            bind:inputDomain
            requestPending={lookup.pending}
            id="domain-input"
            onsubmit={submit}
        />
    {/snippet}

    {#if lookup.pending}
        <div
            class="pt-5 d-flex align-items-center justify-content-center"
            in:fade={{ duration: 200 }}
        >
            <div class="text-center text-muted">
                <Spinner />
                <p class="mt-3">{$t("wait.wait")}…</p>
            </div>
        </div>
    {:else if lookup.notFound}
        <div in:fly={{ y: 20, duration: 350, easing: cubicOut }}>
            {#if watch}
                <div class="mt-3">
                    <WatchCard {watch} showDetails={false} />
                </div>
            {/if}
        </div>
    {:else if lookup.error !== null}
        <div in:fly={{ y: 20, duration: 350, easing: cubicOut }}>
            {#if watch}
                <div class="mt-3">
                    <WatchCard {watch} showDetails={false} />
                </div>
            {/if}
            <h2 class="display-7 fw-bold mt-3">
                <i class="bi bi-exclamation-triangle"></i>
                {$t("domaininfo.error")}
            </h2>
            <div class="card border-danger mt-3">
                <div class="card-body">
                    <div class="d-flex align-items-center">
                        <i class="bi bi-x-circle text-danger fs-3 me-3"></i>
                        <p class="card-text mb-0">{lookup.error}</p>
                    </div>
                </div>
            </div>
        </div>
    {:else if lookup.info !== null}
        <div in:fly={{ y: 20, duration: 400, delay: MORPH_DURATION * 0.6, easing: cubicOut }}>
            {#if watch}
                <div class="mt-3 mb-4">
                    <WatchCard {watch} showDetails={false} />
                </div>
            {/if}
            <DomainInfoDisplay info={lookup.info} {domain} summary={false} />
        </div>
    {/if}
</LookupLayout>
