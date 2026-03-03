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
    import { Button, Spinner } from "@sveltestrap/sveltestrap";
    import hljs from "highlight.js/lib/core";
    import dns from "highlight.js/lib/languages/dns";
    import "highlight.js/styles/github.css";

    import { viewZone as APIViewZone } from "$lib/api/zone";
    import type { Domain } from "$lib/model/domain";
    import { t } from "$lib/translations";

    hljs.registerLanguage("dns", dns);

    interface Props {
        data: { domain: Domain; history: string };
    }

    let { data }: Props = $props();

    let copied = $state(false);
    function copyToClipboard(content: string): void {
        navigator.clipboard.writeText(content).then(() => {
            copied = true;
            setTimeout(() => (copied = false), 2000);
        });
    }

    function highlight(content: string): string {
        return hljs.highlight(content, { language: "dns" }).value;
    }
</script>

<div class="flex-fill pb-1 pt-2 d-flex flex-column" style="min-width: 0;">
    {#await APIViewZone(data.domain, data.history)}
        <h2>{$t("domains.view.title")} <span class="font-monospace">{data.domain.domain}</span></h2>
        <div class="mt-5 text-center">
            <Spinner />
            <p>{$t("wait.formating")}</p>
        </div>
    {:then zoneContent}
        <div class="d-flex align-items-center justify-content-between mb-2">
            <h2 class="mb-0">
                {$t("domains.view.title")} <span class="font-monospace">{data.domain.domain}</span>
            </h2>
            <Button
                color="secondary"
                outline
                size="sm"
                onclick={() => copyToClipboard(zoneContent)}
            >
                {#if copied}
                    <i class="bi bi-clipboard-check"></i>
                {:else}
                    <i class="bi bi-clipboard"></i>
                {/if}
                {$t("common.copy-clipboard")}
            </Button>
        </div>
        <pre class="flex-fill mb-0"><code>{@html highlight(zoneContent)}</code></pre>
    {/await}
</div>
