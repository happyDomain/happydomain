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
    import { Icon, Spinner } from "@sveltestrap/sveltestrap";

    import type { Domain } from "$lib/model/domain";
    import { thisZone } from "$lib/stores/thiszone";
    import { t } from "$lib/translations";
    import {
        buildContext,
        getValidators,
        runAsyncValidators,
        runSyncValidators,
        type ComplianceIssue,
        type Severity,
    } from "$lib/services/compliance";
    // Side-effect import: each validator module self-registers when loaded.
    import "$lib/services/compliance/registry";

    interface Props {
        dn: string;
        origin: Domain;
        type: string;
        value: Record<string, any>;
    }

    let { dn, origin, type, value }: Props = $props();

    const ASYNC_DEBOUNCE_MS = 400;

    let asyncIssues: ComplianceIssue[] = $state([]);
    let asyncRunning = $state(false);
    let abortCtrl: AbortController | null = null;
    let debounceTimer: ReturnType<typeof setTimeout> | null = null;

    let validators = $derived(getValidators(type));
    let registered = $derived(validators !== undefined);
    let hasAsync = $derived(Boolean(validators?.async));

    let ctx = $derived(buildContext(dn, origin, $thisZone));

    let syncIssues = $derived.by<ComplianceIssue[]>(() => {
        if (!registered) return [];
        return runSyncValidators(type, value ?? {}, ctx);
    });

    // A deep snapshot serves as the dependency for the async effect: a plain
    // `void value` only tracks the top-level reference, so deep mutations like
    // `value.txt.Txt = ...` would not retrigger the async validator.
    let valueSnapshot = $derived.by(() => {
        try {
            return JSON.stringify(value ?? {});
        } catch {
            return "";
        }
    });

    $effect(() => {
        if (!hasAsync) {
            asyncIssues = [];
            asyncRunning = false;
            return;
        }
        // Track dependencies so the effect re-runs on changes:
        void valueSnapshot;
        void ctx;

        if (abortCtrl) abortCtrl.abort();
        if (debounceTimer) clearTimeout(debounceTimer);

        const localCtrl = new AbortController();
        abortCtrl = localCtrl;
        asyncRunning = true;

        debounceTimer = setTimeout(async () => {
            const issues = await runAsyncValidators(type, value ?? {}, ctx, localCtrl.signal);
            if (localCtrl.signal.aborted) return;
            asyncIssues = issues;
            asyncRunning = false;
        }, ASYNC_DEBOUNCE_MS);

        return () => {
            localCtrl.abort();
            if (debounceTimer) clearTimeout(debounceTimer);
        };
    });

    let issues = $derived<ComplianceIssue[]>([...syncIssues, ...asyncIssues]);

    const severityOrder: Record<Severity, number> = { error: 0, warning: 1, info: 2 };
    const severityClass: Record<Severity, string> = {
        error: "alert-danger",
        warning: "alert-warning",
        info: "alert-info",
    };
    const severityIcon: Record<Severity, string> = {
        error: "exclamation-octagon-fill",
        warning: "exclamation-triangle-fill",
        info: "info-circle-fill",
    };
    let sortedIssues = $derived(
        [...issues].sort((a, b) => severityOrder[a.severity] - severityOrder[b.severity]),
    );

    function hasDetail(id: string): boolean {
        const key = `compliance.${id}.detail`;
        const txt = $t(key);
        return typeof txt === "string" && txt.length > 0 && txt !== key;
    }
</script>

{#if registered && (sortedIssues.length > 0 || asyncRunning)}
    <section class="mt-3" aria-label={$t("compliance.title")}>
        <h5 class="text-secondary border-bottom border-1 pb-1 d-flex align-items-center gap-2">
            <Icon name="shield-check" />
            {$t("compliance.title")}
            {#if asyncRunning}
                <Spinner size="sm" type="border" color="secondary" />
                <small class="text-muted fst-italic">{$t("compliance.checking")}</small>
            {/if}
        </h5>
        {#if sortedIssues.length > 0}
            <ul class="list-unstyled mb-0">
                {#each sortedIssues as issue (issue.id + (issue.field ?? "") + JSON.stringify(issue.params ?? {}))}
                    <li class="alert {severityClass[issue.severity]} py-2 px-3 mb-2">
                        <div class="d-flex align-items-start gap-2">
                            <Icon name={severityIcon[issue.severity]} />
                            <div class="flex-fill">
                                <div class="fw-semibold">
                                    {$t(`compliance.${issue.id}.title`, issue.params ?? {})}
                                </div>
                                {#if hasDetail(issue.id)}
                                    <div class="small">
                                        {$t(`compliance.${issue.id}.detail`, issue.params ?? {})}
                                    </div>
                                {/if}
                                {#if issue.docUrl}
                                    <a
                                        class="small"
                                        href={issue.docUrl}
                                        target="_blank"
                                        rel="noopener noreferrer"
                                    >
                                        {$t("compliance.learn-more")}
                                        <Icon name="box-arrow-up-right" />
                                    </a>
                                {/if}
                            </div>
                        </div>
                    </li>
                {/each}
            </ul>
        {/if}
    </section>
{/if}
