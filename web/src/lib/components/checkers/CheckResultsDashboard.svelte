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
    import type { Snippet } from "svelte";

    import type { CheckerLinks } from "$lib/checker_links";
    import { Alert, Card, Icon } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import {
        attentionGroups,
        type CheckItem,
        type CheckSection,
        type CheckTarget,
    } from "$lib/domain_checks";
    import { StatusCrit, StatusError, StatusWarn } from "$lib/utils/checkers";
    import CheckGroupCard from "./CheckGroupCard.svelte";
    import CheckTile from "./CheckTile.svelte";
    import DashboardSection from "$lib/components/DashboardSection.svelte";
    import PageTitle from "$lib/components/PageTitle.svelte";

    interface Props {
        domainName: string;
        title: string;
        // Pending while the checks load, or while the zone does.
        sections?: Promise<CheckSection[]>;
        /**
         * Where the checkers of the domain, or of one of its services, live.
         * Handed in because only the route knows: this component is shared.
         */
        linksFor: (target?: CheckTarget) => CheckerLinks;
        /** Actions and facts about the domain, shown beside its name. */
        summary?: Snippet;
        /** Shown after the checks, such as the subdomains of the zone. */
        children?: Snippet;
        /** How many problems to show; all of them when unset. */
        attentionLimit?: number;
        /** Where all the problems are listed, offered past the limit. */
        attentionLink?: string;
    }

    let {
        domainName,
        title,
        sections: sectionsPromise,
        linksFor,
        summary,
        children,
        attentionLimit,
        attentionLink,
    }: Props = $props();

    // The last checks loaded: they stay shown while newer ones load, so
    // coming back to the dashboard doesn't flash a loading screen.
    let loaded: { sections?: CheckSection[]; error?: Error } = $state({});
    $effect(() => {
        const promise = sectionsPromise;
        if (!promise) {
            loaded = {};
            return;
        }
        let current = true;
        promise.then(
            (sections) => {
                if (current) loaded = { sections };
            },
            (error) => {
                if (current) loaded = { error };
            },
        );
        return () => {
            current = false;
        };
    });

    function countCritical(sections: CheckSection[]): number {
        let n = 0;
        for (const s of sections) {
            for (const c of s.items) {
                const code = c.status.latestExecution?.result?.status;
                if (code === StatusCrit || code === StatusError) n++;
            }
        }
        return n;
    }

    function countWarn(sections: CheckSection[]): number {
        let n = 0;
        for (const s of sections) {
            for (const c of s.items) {
                if (c.status.latestExecution?.result?.status === StatusWarn) n++;
            }
        }
        return n;
    }

    // The checks about the domain itself rather than one of its services.
    function domainLevel(sections: CheckSection[]): CheckItem[] {
        return sections.flatMap((s) => s.items.filter((item) => !item.target));
    }
</script>

<svelte:head>
    <title>{title} - HappyDomain</title>
</svelte:head>

<div class="flex-fill mt-1 mb-5">
    <PageTitle classTitle="font-monospace" title={domainName} align="start" children={summary}>
        {#snippet undertitle()}
            {#if loaded.sections}
                {@const sections = loaded.sections}
                {#if sections.some((s) => s.items.length > 0)}
                    {@const critCount = countCritical(sections)}
                    {@const warnCount = countWarn(sections)}
                    {@const okCount =
                        sections.reduce((n, s) => n + s.items.length, 0) - critCount - warnCount}
                    {@const color =
                        critCount > 0 ? "danger" : warnCount > 0 ? "warning" : "success"}
                    <span
                        class="badge rounded-pill d-inline-flex align-items-center gap-2 mt-3 px-3 py-2 fs-6 fw-semibold border bg-{color}-subtle text-{color}-emphasis border-{color}-subtle"
                    >
                        <span class="d-inline-block p-1 rounded-circle bg-{color}"></span>
                        {[
                            critCount > 0 && `${critCount} ${$t("checkers.dashboard.critical")}`,
                            warnCount > 0 &&
                                `${warnCount} ${$t("checkers.dashboard.needs-attention")}`,
                            okCount > 0 && `${okCount} ${$t("checkers.dashboard.passing")}`,
                        ]
                            .filter(Boolean)
                            .join(" · ")}
                    </span>
                {/if}
            {/if}
        {/snippet}
    </PageTitle>

    {#snippet loading()}
        <Card body>
            <p class="text-center mb-0">
                <span class="spinner-border spinner-border-sm me-2"></span>
                {$t("checkers.list.loading")}
            </p>
        </Card>
    {/snippet}

    {#if loaded.error}
        <Alert color="danger">
            <Icon name="exclamation-triangle-fill" />
            {$t("checkers.list.error-loading", { error: loaded.error.message })}
        </Alert>
    {:else if !loaded.sections}
        {@render loading()}
    {:else}
        {@const sections = loaded.sections}
        {#if sections.every((s) => s.items.length === 0)}
            <Alert color="info">
                <Icon name="info-circle" />
                {$t("checkers.list.no-checks")}
            </Alert>
        {:else}
            {@const attention = attentionGroups(sections)}
            {@const overview = domainLevel(sections)}

            {#if attention.length > 0}
                {@const shown = attentionLimit ? attention.slice(0, attentionLimit) : attention}
                <DashboardSection
                    title={$t("domains.dashboard.attention.title")}
                    subtitle={$t("domains.dashboard.attention.subtitle", {
                        count: attention.length,
                    })}
                >
                    <div class="vstack gap-3">
                        {#each shown as group (group.status + group.checkerId)}
                            <CheckGroupCard {group} {linksFor} />
                        {/each}
                        {#if shown.length < attention.length && attentionLink}
                            <a
                                href={attentionLink}
                                class="btn btn-outline-secondary align-self-center"
                            >
                                {$t("domains.dashboard.attention.see-all", {
                                    count: attention.length,
                                })}
                            </a>
                        {/if}
                    </div>
                </DashboardSection>
            {:else}
                <section class="card card-body flex-row align-items-center gap-3 mt-5">
                    <span
                        class="d-flex flex-shrink-0 p-2 lh-1 rounded-circle bg-success-subtle text-success-emphasis fs-5"
                    >
                        <Icon name="check-lg" />
                    </span>
                    <div>
                        <h2 class="fs-5 fw-semibold text-body-emphasis mb-0">
                            {$t("domains.dashboard.clean.title")}
                        </h2>
                        <p class="small text-body-secondary mt-1 mb-0">
                            {$t("domains.dashboard.clean.note", {
                                count: sections.reduce((n, s) => n + s.items.length, 0),
                            })}
                        </p>
                    </div>
                </section>
            {/if}

            {#if overview.length > 0}
                <DashboardSection
                    title={$t("domains.dashboard.health.title")}
                    subtitle={$t("domains.dashboard.health.subtitle")}
                >
                    <div class="row row-cols-1 row-cols-md-2 row-cols-xl-3 g-3">
                        {#each overview as item}
                            {@const checker = item.status}
                            {@const links = linksFor(item.target)}
                            {@const exec = checker.latestExecution}
                            <div class="col">
                                <CheckTile
                                    status={exec?.result?.status}
                                    name={checker.name || checker.id || ""}
                                    message={exec?.messages?.[0]}
                                    startedAt={exec?.startedAt}
                                    href={exec?.id
                                        ? links.execution!(checker.id!, exec.id!)
                                        : links.checker(checker.id!)}
                                />
                            </div>
                        {/each}
                    </div>
                </DashboardSection>
            {/if}
        {/if}
    {/if}

    {@render children?.()}
</div>
