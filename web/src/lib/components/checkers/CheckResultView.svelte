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
    import { Alert, Spinner, Table } from "@sveltestrap/sveltestrap";

    import { onDestroy } from "svelte";
    import { t } from "$lib/translations";
    import type { CheckerInfo, CheckResult, MetricsReport } from "$lib/model/checker";
    import {
        currentCheckResult,
        currentCheckInfo,
        showHTMLReport,
        reportViewMode,
    } from "$lib/stores/checkers";

    interface Props {
        resultPromise: Promise<CheckResult>;
        checkPromise: Promise<CheckerInfo>;
        htmlReportPromise: Promise<string>;
        getMetrics: () => Promise<MetricsReport>;
    }

    let { resultPromise, checkPromise, htmlReportPromise, getMetrics }: Props = $props();

    let metricsReport = $state<MetricsReport | null>(null);

    $effect(() => {
        resultPromise.then((r) => currentCheckResult.set(r));
    });

    $effect(() => {
        metricsReport = null;
        checkPromise.then((c) => {
            currentCheckInfo.set(c);
            if (c.has_metrics) {
                reportViewMode.set("metrics");
                getMetrics()
                    .then((r) => (metricsReport = r))
                    .catch(() => {});
            }
        });
    });

    onDestroy(() => {
        currentCheckResult.set(null);
        currentCheckInfo.set(null);
        showHTMLReport.set(true);
        reportViewMode.set("html");
    });
</script>

<div class="flex-fill mw-100 d-flex flex-column">
    {#await Promise.all([resultPromise, checkPromise])}
        <div class="mt-5 text-center flex-fill">
            <Spinner />
            <p>{$t("checkers.result.loading")}</p>
        </div>
    {:then [result, check]}
        {#if result.report || check.has_html_report || check.has_metrics}
            {#if check.has_metrics && $reportViewMode === "metrics"}
                <div class="p-3 flex-fill">
                    {#if metricsReport}
                        <Table size="sm" hover striped>
                            <thead>
                                <tr>
                                    <th>Metric</th>
                                    <th class="text-end">Value</th>
                                    <th>Unit</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each metricsReport.series as series}
                                    {#each series.points as point}
                                        <tr>
                                            <td>{series.label}</td>
                                            <td class="text-end font-monospace">{point.value}</td>
                                            <td class="text-muted">{series.unit}</td>
                                        </tr>
                                    {/each}
                                {/each}
                            </tbody>
                        </Table>
                    {:else}
                        <div class="text-center p-4"><Spinner /></div>
                    {/if}
                </div>
            {:else if check.has_html_report && ($reportViewMode === "html" || ($showHTMLReport && $reportViewMode !== "json"))}
                {#await htmlReportPromise}
                    <div class="text-center p-4"><Spinner /></div>
                {:then html}
                    <iframe
                        srcdoc={html}
                        sandbox=""
                        title={$t("checkers.result.full-report")}
                        class="flex-fill"
                        style="width: 100%; border: none; display: block;"
                    ></iframe>
                {:catch}
                    <pre class="bg-light p-3 rounded mb-0"><code
                            >{JSON.stringify(result.report, null, 2)}</code
                        ></pre>
                {/await}
            {:else if typeof result.report === "string"}
                <pre class="bg-light p-3 rounded mb-0"><code>{result.report}</code></pre>
            {:else}
                <pre class="bg-light p-3 rounded mb-0"><code
                        >{JSON.stringify(result.report, null, 2)}</code
                    ></pre>
            {/if}
        {/if}
    {:catch error}
        <Alert color="danger" class="m-3">
            {$t("checkers.result.error-loading", { error: error.message })}
        </Alert>
    {/await}
</div>
