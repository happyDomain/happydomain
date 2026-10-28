<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2024 happyDomain
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
    import { get } from "svelte/store";
    import { goto } from "$app/navigation";
    import { resolve } from "$app/paths";

    import { Alert, Progress } from "@sveltestrap/sveltestrap";

    import PageTitle from "$lib/components/PageTitle.svelte";
    import TileLoader from "$lib/components/TileLoader.svelte";
    import { getRetrieveZoneProgress, type RetrieveZoneProgress } from "$lib/api/zone";
    import type { Domain } from "$lib/model/domain";
    import { retrieveZone } from "$lib/stores/thiszone";
    import { domains_idx, refreshDomains } from "$lib/stores/domains";
    import { t } from "$lib/translations";

    interface Props {
        data: { domain: Domain };
    }

    let { data }: Props = $props();

    const pollInterval = 1000;

    // A step telling nothing of its own progress would freeze the bar: it
    // creeps towards, but never reaches, the end of the step, like the
    // navigation progress bar does.
    const creepInterval = 200;
    const creepShare = 0.9;
    const creepTau = 5000;
    const reducedMotion =
        typeof window !== "undefined" &&
        window.matchMedia("(prefers-reduced-motion: reduce)").matches;

    let progress = $state<RetrieveZoneProgress | null>(null);
    let now = $state(Date.now());
    let finished = $state(false);
    let mainError = $state<unknown>(null);

    // The import outlives the request asking for it, which a proxy may cut
    // short: once the import was seen in flight, the failure of the request
    // is only told when the import ends without a zone.
    let seenInFlight = false;
    let requestError: unknown = null;

    // shown never goes back; since is when the server's percent last moved.
    let shown = $state(0);
    let since = Date.now();

    let pollTimer: ReturnType<typeof setInterval> | undefined;
    let creepTimer: ReturnType<typeof setInterval> | undefined;

    function stopPolling(): void {
        clearInterval(pollTimer);
        clearInterval(creepTimer);
        pollTimer = creepTimer = undefined;
    }

    function creep(): void {
        now = Date.now();

        const percent = progress?.percent ?? 0;
        const gap = Math.max(0, (progress?.step_end ?? 0) - percent);
        const crept = reducedMotion
            ? 0
            : gap * creepShare * (1 - Math.exp(-(now - since) / creepTau));
        shown = Math.max(shown, percent + crept);
    }

    function hasZone(): boolean {
        return !!get(domains_idx)[data.domain.domain]?.zone_history?.length;
    }

    async function done(): Promise<void> {
        stopPolling();
        finished = true;
        await refreshDomains();
        goto(
            resolve("/domains/[dn]", {
                dn: encodeURIComponent(
                    get(domains_idx)[data.domain.domain] ? data.domain.domain : data.domain.id,
                ),
            }),
            { replaceState: true },
        );
    }

    function fail(err: unknown): void {
        stopPolling();
        mainError = err;
    }

    async function poll(): Promise<void> {
        let current: RetrieveZoneProgress | null;
        try {
            current = await getRetrieveZoneProgress(data.domain);
        } catch (err) {
            console.warn("Failed to poll the progress of the import", err);
            return;
        }

        if (current) {
            if (current.percent !== progress?.percent) {
                since = Date.now();
            }
            progress = current;
            seenInFlight = true;
            creep();
        } else if (requestError) {
            // The import ended after its request failed.
            stopPolling();
            await refreshDomains();
            if (hasZone()) {
                done();
            } else {
                fail(requestError);
            }
        }
    }

    onMount(() => {
        pollTimer = setInterval(poll, pollInterval);
        creepTimer = setInterval(creep, creepInterval);

        retrieveZone(data.domain).then(done, (err) => {
            if (seenInFlight) {
                requestError = err;
            } else {
                fail(err);
            }
        });

        return stopPolling;
    });

    let elapsed = $derived(
        progress?.started_at
            ? Math.max(0, Math.round((now - new Date(progress.started_at).getTime()) / 1000))
            : 0,
    );
    let total = $derived(progress?.total ?? 0);
    // A provider retrieving the zone tells nothing of its progress.
    let indeterminate = $derived(progress?.step === "retrieving");
    let percent = $derived(indeterminate ? 100 : shown);
</script>

<div class="flex-fill d-flex flex-column pb-5">
    <PageTitle
        title={$t("zones.retrieve")}
        domain={data.domain.domain}
        subtitle={$t("zones.retrieve-subtitle")}
    />
    {#if mainError}
        <div class="mt-4 text-center flex-fill">
            <Alert color="danger" fade={false}>
                <strong>{$t("errors.domain-import")}</strong>
                {mainError}
            </Alert>
        </div>
    {:else if finished}
        <div class="mt-4 text-center flex-fill">
            <TileLoader class="mb-2" />
            <p>{$t("wait.wait")}</p>
        </div>
    {:else}
        <div class="mt-4 mx-auto flex-fill" style="max-width: 40rem; width: 100%">
            {#if progress}
                <p class="mb-2">
                    {$t("zones.progress." + progress.step, { source: progress.source ?? "" })}
                </p>
                <Progress class="mb-2" value={percent} color="primary" striped animated />
                <p class="text-muted small d-flex flex-wrap justify-content-between gap-2">
                    <span>
                        {#if total > 0}
                            {$t("zones.progress.names", { done: progress.done ?? 0, total })},
                        {/if}
                        {$t("zones.progress.records", { count: progress.records ?? 0 })}
                        {#if progress.queries}
                            ({$t("zones.progress.queries", { count: progress.queries })})
                        {/if}
                    </span>
                    <span>{$t("zones.progress.elapsed", { count: elapsed })}</span>
                </p>
            {:else}
                <div class="text-center">
                    <TileLoader class="mb-2" />
                    <p>{$t("wait.importing")}</p>
                </div>
            {/if}
            <p class="text-muted text-center mt-4">
                {$t("zones.progress.can-leave")}
            </p>
        </div>
    {/if}
</div>
