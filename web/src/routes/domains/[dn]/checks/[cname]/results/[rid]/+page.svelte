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
    import { Alert, Spinner } from "@sveltestrap/sveltestrap";

    import { onDestroy } from "svelte";
    import { t } from "$lib/translations";
    import { page } from "$app/state";
    import {
        getCheckStatus,
        getCheckResult,
        getCheckResultHTMLReport,
    } from "$lib/api/checks";
    import type { Domain } from "$lib/model/domain";
    import { currentCheckResult, currentCheckInfo, showHTMLReport } from "$lib/stores/checks";

    interface Props {
        data: { domain: Domain };
    }

    let { data }: Props = $props();

    const checkName = $derived(page.params.cname || "");
    const resultId = $derived(page.params.rid || "");

    let resultPromise = $derived(getCheckResult(data.domain.id, checkName, resultId));
    let checkPromise = $derived(getCheckStatus(checkName));
    let htmlReportPromise = $derived(getCheckResultHTMLReport(data.domain.id, checkName, resultId));

    $effect(() => {
        resultPromise.then((r) => currentCheckResult.set(r));
    });

    $effect(() => {
        checkPromise.then((c) => currentCheckInfo.set(c));
    });

    onDestroy(() => {
        currentCheckResult.set(null);
        currentCheckInfo.set(null);
        showHTMLReport.set(true);
    });
</script>

<svelte:head>
    <title>
        Check Result - {checkName} - {data.domain.domain} - happyDomain
    </title>
</svelte:head>

<div class="flex-fill mw-100 d-flex flex-column">
    {#await Promise.all([resultPromise, checkPromise])}
        <div class="mt-5 text-center flex-fill">
            <Spinner />
            <p>{$t("checks.result.loading")}</p>
        </div>
    {:then [result, check]}
        {#if result.report || check.has_html_report}
            {#if check.has_html_report && $showHTMLReport}
                {#await htmlReportPromise}
                    <div class="text-center p-4"><Spinner /></div>
                {:then html}
                    <iframe
                        srcdoc={html}
                        sandbox=""
                        title={$t("checks.result.full-report")}
                        class="flex-fill"
                        style="width: 100%; border: none; display: block;"
                    ></iframe>
                {:catch}
                    <pre class="bg-light p-3 rounded mb-0"><code>{JSON.stringify(result.report, null, 2)}</code></pre>
                {/await}
            {:else if typeof result.report === "string"}
                <pre class="bg-light p-3 rounded mb-0"><code>{result.report}</code></pre>
            {:else}
                <pre class="bg-light p-3 rounded mb-0"><code>{JSON.stringify(result.report, null, 2)}</code></pre>
            {/if}
        {/if}
    {:catch error}
        <Alert color="danger" class="m-3">
            {$t("checks.result.error-loading", { error: error.message })}
        </Alert>
    {/await}
</div>
