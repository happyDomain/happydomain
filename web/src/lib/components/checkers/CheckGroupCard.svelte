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
    import type { CheckerLinks } from "$lib/checker_links";

    import { t } from "$lib/translations";
    import type { AttentionGroup, CheckItem, CheckTarget } from "$lib/domain_checks";
    import { getStatusColor, getStatusI18nKey } from "$lib/utils";
    import CheckCard from "./CheckCard.svelte";

    interface Props {
        group: AttentionGroup;
        linksFor: (target?: CheckTarget) => CheckerLinks;
    }

    let { group, linksFor }: Props = $props();

    // The services shown before the rest is unfolded.
    const FOLDED = 3;

    let unfolded = $state(false);
    let color = $derived(getStatusColor(group.status));
    let first = $derived(group.items[0]);
    let shown = $derived(unfolded ? group.items : group.items.slice(0, FOLDED));

    function chip(item: CheckItem): string | undefined {
        if (!item.target) return undefined;
        return item.chip ? `${item.chip} · ${item.target.subdomain}` : item.target.subdomain;
    }

    function detailsLink(item: CheckItem): string {
        const links = linksFor(item.target);
        const exec = item.status.latestExecution;
        return exec?.id
            ? links.execution!(item.status.id!, exec.id)
            : links.checker(item.status.id!);
    }
</script>

{#if group.items.length === 1}
    {@const checker = first.status}
    {@const links = linksFor(first.target)}
    {@const exec = checker.latestExecution}
    <CheckCard
        status={exec?.result?.status}
        name={checker.name || checker.id || ""}
        chip={chip(first)}
        configureLink={links.checker(checker.id!)}
        detailsLink={exec?.id ? links.execution!(checker.id!, exec.id!) : undefined}
        historyLink={links.executions!(checker.id!)}
        messages={exec?.messages}
        startedAt={exec?.startedAt}
        scope={first.scope}
        checkerId={checker.id}
        hasMetrics={checker.has_metrics}
    />
{:else}
    <article class="card overflow-hidden border-{color}-subtle">
        <header
            class="card-header d-flex align-items-center gap-2 py-2 small fw-semibold bg-{color}-subtle border-{color}-subtle"
        >
            <span
                class="d-inline-flex align-items-center gap-2 text-uppercase text-{color}-emphasis"
            >
                <span class="d-inline-block p-1 rounded-circle bg-{color}"></span>
                {$t(getStatusI18nKey(group.status))}
            </span>
            <span class="badge rounded-pill bg-body-secondary text-body-secondary fw-normal">
                {$t("domains.n-services", { count: group.items.length })}
            </span>
        </header>
        <div class="card-body vstack gap-2">
            <h5 class="fs-6 fw-semibold text-body-emphasis mb-0">
                {first.status.name || group.checkerId}
            </h5>
            <ul class="list-unstyled vstack gap-1 small mb-0">
                {#each shown as item}
                    {@const exec = item.status.latestExecution}
                    <li class="d-flex align-items-baseline gap-2 min-w-0">
                        <span
                            class="d-inline-block p-1 rounded-circle flex-shrink-0 bg-{getStatusColor(
                                exec?.result?.status,
                            )}"
                        ></span>
                        <a
                            href={detailsLink(item)}
                            class="font-monospace text-nowrap link-body-emphasis link-offset-1 link-underline-opacity-25 link-underline-opacity-100-hover"
                        >
                            {chip(item) ?? $t("domains.dashboard.attention.the-domain")}
                        </a>
                        {#if exec?.messages?.[0]}
                            <span class="text-body-secondary text-truncate min-w-0">
                                {exec.messages[0]}
                            </span>
                        {/if}
                    </li>
                {/each}
            </ul>
            {#if group.items.length > FOLDED}
                <button
                    type="button"
                    class="btn btn-link btn-sm p-0 align-self-start link-secondary"
                    onclick={() => (unfolded = !unfolded)}
                >
                    {unfolded
                        ? $t("domains.dashboard.attention.fold")
                        : $t("domains.dashboard.attention.unfold", {
                              count: group.items.length - FOLDED,
                          })}
                </button>
            {/if}
        </div>
    </article>
{/if}
