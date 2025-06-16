<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2024 happyDomain
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
    import { createEventDispatcher } from "svelte";

    import { Icon, Spinner } from "@sveltestrap/sveltestrap";

    import { diffZone as APIDiffZone } from "$lib/api/zone";
    import type { Correction } from "$lib/model/correction";
    import type { Domain } from "$lib/model/domain";
    import { t } from "$lib/translations";

    const dispatch = createEventDispatcher();

    export let domain: Domain;
    export let zoneFrom: string;
    export let zoneTo: string;
    export let selectable = false;
    export let selectedDiff: Array<string> | null = null;

    let zoneDiff: Array<{ className: string; msg: string; id: string }>;
    $: computeDiff(domain, zoneTo, zoneFrom);

    const correctionsIdx: Record<string, Correction> = {};

    function computeDiff(domain: Domain, zoneTo: string, zoneFrom: string) {
        APIDiffZone(domain, zoneTo, zoneFrom).then(
            (v: Array<Correction>) => {
                let zoneDiffCreated = 0;
                let zoneDiffDeleted = 0;
                let zoneDiffModified = 0;
                zoneDiff = [];
                selectedDiff = [];
                if (v) {
                    for (const c of v) {
                        if (!c.id) return;

                        correctionsIdx[c.id] = c;

                        let className = "";
                        if (c.kind == 1) {
                            className = "text-success";
                            zoneDiffCreated += 1;
                        } else if (c.kind == 2) {
                            className = "text-warning";
                            zoneDiffModified += 1;
                        } else if (c.kind == 3) {
                            className = "text-danger";
                            zoneDiffDeleted += 1;
                        } else if (c.kind == 99) {
                            className = "text-info";
                        }

                        zoneDiff.push({
                            className,
                            msg: c.msg,
                            id: c.id,
                        });
                        selectedDiff.push(c.id)
                    }
                }
                dispatch("computed-diff", {
                    zoneDiffLength: zoneDiff.length,
                    zoneDiffCreated,
                    zoneDiffDeleted,
                    zoneDiffModified,
                });
                dispatchSelectionSummary();
            },
            (err: any) => {
                dispatch("error", err);
            },
        );
    }

    function dispatchSelectionSummary() {
        dispatch("computed-selection", {
            selectedDiffCreated: !selectedDiff
                ? 0
                : selectedDiff.filter((id: string) => correctionsIdx[id].kind == 1).length,
            selectedDiffDeleted: !selectedDiff
                ? 0
                : selectedDiff.filter((id: string) => correctionsIdx[id].kind == 3).length,
            selectedDiffModified: !selectedDiff
                ? 0
                : selectedDiff.filter((id: string) => correctionsIdx[id].kind == 2).length,
        });
    }
</script>

{#if !zoneDiff}
    <div class="my-2 text-center">
        <Spinner color="warning" />
        <p>{$t("wait.exporting")}</p>
    </div>
{:else if zoneDiff.length == 0}
    <slot name="nodiff">Aucune différence.</slot>
{:else}
    {#each zoneDiff as line, n}
        <div
            class={"font-monospace " + line.className}
            class:col={selectable}
            class:form-check={selectable}
        >
            {#if selectable}
                <input
                    type="checkbox"
                    class="form-check-input"
                    id="correction-{line.id}"
                    bind:group={selectedDiff}
                    value={line.id}
                    on:change={dispatchSelectionSummary}
                />
                <label
                    class="form-check-label"
                    for="correction-{line.id}"
                    style="padding-left: 1em; text-indent: -1em;"
                >
                    {line.msg}
                </label>
            {:else}
                {line.msg}
            {/if}
        </div>
    {/each}
{/if}
