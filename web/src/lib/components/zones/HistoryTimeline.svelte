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

<script module lang="ts">
    import type { Domain } from "$lib/model/domain";

    // The most recent published version is the one live on the provider.
    export function liveVersion(domain: Domain): string | undefined {
        return domain.zone_history?.find((zid) => domain.zone_meta?.[zid]?.published);
    }
</script>

<script lang="ts">
    import { resolve } from "$app/paths";

    import { Badge, Icon } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";

    interface Props {
        domain: Domain;
        selected?: string;
        /** The domain as it appears in URLs, already encoded */
        dn: string;
    }

    let { domain, selected, dn }: Props = $props();

    let live = $derived(liveVersion(domain));

    const monthFmt = new Intl.DateTimeFormat(undefined, { month: "long", year: "numeric" });
    const entryFmt = new Intl.DateTimeFormat(undefined, {
        day: "numeric",
        month: "short",
        hour: "2-digit",
        minute: "2-digit",
    });

    let groups = $derived.by(() => {
        const res: Array<{ month: string; zids: string[] }> = [];
        for (const zid of domain.zone_history ?? []) {
            const meta = domain.zone_meta?.[zid];
            if (!meta) continue;
            const month = monthFmt.format(new Date(meta.last_modified));
            if (res.length === 0 || res[res.length - 1].month !== month) {
                res.push({ month, zids: [] });
            }
            res[res.length - 1].zids.push(zid);
        }
        return res;
    });

    let containerEl: HTMLElement | null = $state(null);
    $effect(() => {
        if (selected && containerEl) {
            containerEl
                .querySelector('[aria-current="page"]')
                ?.scrollIntoView({ block: "nearest", behavior: "smooth" });
        }
    });
</script>

<div bind:this={containerEl}>
    {#each groups as group (group.month)}
        <div class="timeline-month text-muted small fw-semibold text-uppercase mt-3 mb-1">
            {group.month}
        </div>
        {#each group.zids as zid (zid)}
            {@const meta = domain.zone_meta![zid]}
            <a
                href={resolve("/domains/[dn]/history/[zid]", {
                    dn,
                    zid: encodeURIComponent(zid),
                })}
                class="timeline-entry d-block text-decoration-none text-body"
                class:fw-bold={zid === selected}
                class:bg-primary-subtle={zid === selected}
                class:timeline-success={zid === live}
                aria-current={zid === selected ? "page" : undefined}
            >
                <span class="d-flex align-items-center gap-1">
                    <span class="text-truncate">
                        {entryFmt.format(new Date(meta.last_modified))}
                    </span>
                    {#if zid === domain.zone_history[0] && !meta.published}
                        <Badge color="secondary" class="fw-normal">
                            {$t("history.draft")}
                        </Badge>
                    {:else if zid === live}
                        <Badge color="success" class="fw-normal">{$t("history.live")}</Badge>
                    {:else if meta.published}
                        <Icon
                            name="cloud-upload"
                            class="text-muted"
                            title={$t("history.published-on")}
                        />
                    {/if}
                    <img
                        src={"/api/users/" + encodeURIComponent(meta.id_author) + "/avatar.png"}
                        alt=""
                        class="timeline-avatar ms-auto rounded-1"
                    />
                </span>
                {#if meta.commit_message}
                    <span class="d-block small text-muted text-truncate">
                        {meta.commit_message}
                    </span>
                {/if}
            </a>
        {/each}
    {/each}
</div>

<style>
    .timeline-avatar {
        height: 1.1em;
    }
</style>
