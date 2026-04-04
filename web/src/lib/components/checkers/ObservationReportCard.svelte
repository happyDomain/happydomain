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
    import { Spinner } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import type { CheckerScope, CheckMetric, ObservationSnapshotWithData } from "$lib/api/checkers";
    import { getScopedExecutionHTMLReport } from "$lib/api/checkers";
    import { showHTMLReport, cachedHTMLReport } from "$lib/stores/checkers";

    interface Props {
        observations: ObservationSnapshotWithData;
        metrics?: CheckMetric[] | null;
        scope?: CheckerScope;
        checkerId?: string;
        execId?: string;
    }

    let { observations, scope, checkerId, execId }: Props = $props();

    let htmlReportPromise = $state<Promise<string> | null>(null);

    $effect(() => {
        if ($showHTMLReport && scope && checkerId && execId && observations?.data) {
            const keys = Object.keys(observations.data);
            if (keys.length > 0) {
                const promise = getScopedExecutionHTMLReport(scope, checkerId, execId, keys[0]);
                promise.then((html) => cachedHTMLReport.set(html)).catch(() => {});
                htmlReportPromise = promise;
            }
        }
    });
</script>

{#if observations?.data && Object.keys(observations.data).length > 0}
    <div
        class="flex-fill d-flex"
        style="overflow: auto; padding: var(--bs-card-spacer-y) var(--bs-card-spacer-x)"
    >
        {#if $showHTMLReport && htmlReportPromise}
            {#await htmlReportPromise}
                <div class="text-center p-4"><Spinner /></div>
            {:then html}
                <iframe
                    srcdoc={html}
                    sandbox=""
                    title={$t("checkers.result.full-report")}
                    style="width: 100%; min-height: 600px; border: none; display: block;"
                ></iframe>
            {:catch}
                <pre class="mb-0" style="width: 0; min-width: 100%"><code
                        >{JSON.stringify(observations.data, null, 2)}</code
                    ></pre>
            {/await}
        {:else}
            <pre class="mb-0" style="width: 0; min-width: 100%"><code
                    >{JSON.stringify(observations.data, null, 2)}</code
                ></pre>
        {/if}
    </div>
{/if}
