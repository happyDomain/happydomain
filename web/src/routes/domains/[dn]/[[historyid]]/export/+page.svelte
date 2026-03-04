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
    import PageTitle from "$lib/components/PageTitle.svelte";
    import type { Domain } from "$lib/model/domain";
    import { t } from "$lib/translations";

    hljs.registerLanguage("dns", dns);

    interface Props {
        data: { domain: Domain; history: string };
    }

    let { data }: Props = $props();

    let zoneContent = $state<string | null>(null);
    let copied = $state(false);

    $effect(() => {
        zoneContent = null;
        APIViewZone(data.domain, data.history).then((content) => {
            zoneContent = content;
        });
    });

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
    <PageTitle
        title={$t("domains.view.title")}
        domain={data.domain.domain}
        subtitle={$t("domains.view.subtitle")}
    >
        <div class="flex-fill d-flex flex-column justify-content-end mb-2">
            <Button
                color="secondary"
                outline
                size="sm"
                onclick={() => zoneContent && copyToClipboard(zoneContent)}
                disabled={!zoneContent}
            >
                {#if copied}
                    <i class="bi bi-clipboard-check"></i>
                {:else}
                    <i class="bi bi-clipboard"></i>
                {/if}
                {$t("common.copy-clipboard")}
            </Button>
        </div>
    </PageTitle>
    {#if zoneContent === null}
        <div class="mt-5 text-center">
            <Spinner />
            <p>{$t("wait.formating")}</p>
        </div>
    {:else}
        <pre class="flex-fill mb-0"><code>{@html highlight(zoneContent)}</code></pre>
    {/if}
</div>
