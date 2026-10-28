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

    import { Badge, Button, Icon } from "@sveltestrap/sveltestrap";

    import DiffZone from "$lib/components/zones/DiffZone.svelte";
    import { liveVersion } from "$lib/components/zones/HistoryTimeline.svelte";
    import PageTitle from "$lib/components/PageTitle.svelte";
    import type { Domain } from "$lib/model/domain";
    import { domainLink } from "$lib/stores/domains";
    import { isReadOnlyProvider, providers_idx } from "$lib/stores/providers";
    import { getUser } from "$lib/stores/users";
    import { t } from "$lib/translations";
    import { controls as ctrlDiffZone } from "../../ModalDiffZone.svelte";

    interface Props {
        data: { domain: Domain; historyDomain: Domain; zid: string };
    }

    let { data }: Props = $props();

    let history = $derived(data.historyDomain.zone_history ?? []);
    let meta = $derived(data.historyDomain.zone_meta![data.zid]);
    let idx = $derived(history.indexOf(data.zid));
    let live = $derived(liveVersion(data.historyDomain));

    // Rolling back to the latest version is just publishing it, and a
    // read-only provider refuses both.
    let canRollback = $derived(
        idx > 0 && !isReadOnlyProvider($providers_idx, data.domain.id_provider),
    );

    // Compare with the previous version by default; the user can pick
    // any other one, or the zone currently at the provider ("@").
    let compareWith: string = $derived(history[idx + 1] ?? "@");

    const longFmt = new Intl.DateTimeFormat(undefined, { dateStyle: "long", timeStyle: "short" });
    const shortFmt = new Intl.DateTimeFormat(undefined, {
        dateStyle: "medium",
        timeStyle: "short",
    });

    function versionLabel(zid: string): string {
        const m = data.historyDomain.zone_meta?.[zid];
        if (!m) return zid;
        let label = shortFmt.format(new Date(m.last_modified));
        if (m.commit_message) label += " · " + m.commit_message;
        if (zid === live) label += " (" + $t("history.live") + ")";
        return label;
    }
</script>

<div class="flex-fill pb-4 pt-2">
    <PageTitle
        title={$t("history.title")}
        domain={data.domain.domain}
        subtitle={$t("history.subtitle")}
    />

    <section class="mb-4">
        <div class="d-flex align-items-center gap-2 flex-wrap">
            <h2 class="h4 mb-0">{longFmt.format(new Date(meta.last_modified))}</h2>
            {#if idx === 0 && !meta.published}
                <Badge color="secondary">{$t("history.draft")}</Badge>
            {:else if data.zid === live}
                <Badge color="success">{$t("history.live")}</Badge>
            {/if}
            <Button
                color="primary"
                outline
                size="sm"
                class="ms-auto"
                href={resolve("/domains/[dn]/history/[zid]/zone", {
                    dn: encodeURIComponent(domainLink(data.domain.id)),
                    zid: encodeURIComponent(data.zid),
                })}
            >
                <Icon name="eye-fill" />
                {$t("history.see")}
            </Button>
            <Button
                color="secondary"
                outline
                size="sm"
                href={resolve("/domains/[dn]/history/[zid]/export", {
                    dn: encodeURIComponent(domainLink(data.domain.id)),
                    zid: encodeURIComponent(data.zid),
                })}
            >
                <Icon name="file-earmark-text" />
                {$t("domains.view.title")}
            </Button>
            {#if canRollback}
                <Button
                    color="warning"
                    size="sm"
                    title={$t("domains.actions.rollback")}
                    on:click={() => ctrlDiffZone.Open()}
                >
                    <Icon name="arrow-counterclockwise" />
                    {$t("domains.actions.rollback")}
                </Button>
            {/if}
        </div>

        <div class="d-flex align-items-center gap-2 mt-2">
            <img
                src={"/api/users/" + encodeURIComponent(meta.id_author) + "/avatar.png"}
                alt=""
                class="rounded-1"
                style="height: 1.5em"
            />
            {#await getUser(meta.id_author) then user}
                <span class="text-muted">{user.email}</span>
            {:catch}
                <!-- The author may no longer exist -->
            {/await}
        </div>

        {#if meta.commit_message}
            <p class="mt-2 mb-0 fst-italic">{meta.commit_message}</p>
        {/if}

        <ul class="list-inline small text-muted mt-2 mb-0">
            {#if meta.published}
                <li class="list-inline-item">
                    <Icon name="cloud-upload" />
                    {$t("history.published-on")}
                    {shortFmt.format(new Date(meta.published))}
                </li>
            {/if}
            {#if meta.commit_date}
                <li class="list-inline-item">
                    <Icon name="bookmark-check" />
                    {$t("history.committed-on")}
                    {shortFmt.format(new Date(meta.commit_date))}
                </li>
            {/if}
        </ul>
    </section>

    <section>
        <div class="d-flex align-items-center gap-2 flex-wrap mb-3">
            <label for="history-compare" class="fw-semibold">{$t("history.compare-with")}</label>
            <select
                id="history-compare"
                class="form-select form-select-sm w-auto mw-100"
                bind:value={compareWith}
            >
                {#each history as zid (zid)}
                    {#if zid !== data.zid}
                        <option value={zid}>{versionLabel(zid)}</option>
                    {/if}
                {/each}
                <option value="@">{$t("history.provider-zone")}</option>
            </select>
        </div>

        {#key data.zid + "/" + compareWith}
            <DiffZone domain={data.historyDomain} zoneFrom={data.zid} zoneTo={compareWith}>
                {#snippet nodiff()}
                    <p class="text-center text-muted my-4">
                        <Icon name="check2-all" class="text-success" />
                        {$t("history.no-diff")}
                    </p>
                {/snippet}
            </DiffZone>
        {/key}
    </section>
</div>
