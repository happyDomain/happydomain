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

    import { Spinner } from "@sveltestrap/sveltestrap";

    import DiffZoneView from "./DiffZoneView.svelte";
    import type { Correction, FullCorrection } from "$lib/model/correction";
    import { getCachedDiffZone } from "$lib/stores/zonediff";
    import type { Domain } from "$lib/model/domain";
    import { t } from "$lib/translations";

    const dispatch = createEventDispatcher();

    interface Props {
        domain: Domain;
        zoneFrom: string;
        zoneTo: string;
        selectable?: boolean;
        disabled?: boolean;
        selectedDiff?: Array<string> | null;
        nodiff?: import("svelte").Snippet;
    }

    let {
        domain,
        zoneFrom,
        zoneTo,
        selectable = false,
        disabled = false,
        selectedDiff = $bindable(null),
        nodiff,
    }: Props = $props();

    let zoneDiff: Array<FullCorrection> | null = $state(null);

    const correctionsIdx: Record<string, Correction> = {};

    function computeDiff(domain: Domain, zoneTo: string, zoneFrom: string) {
        getCachedDiffZone(domain, zoneTo, zoneFrom).then(
            (v: Array<Correction>) => {
                let zoneDiffCreated = 0;
                let zoneDiffDeleted = 0;
                let zoneDiffModified = 0;
                zoneDiff = [];
                selectedDiff = [];
                if (v) {
                    for (const c of v) {
                        if (!c.id || c.kind === undefined) return;

                        correctionsIdx[c.id] = c;

                        if (c.kind == 1) {
                            zoneDiffCreated += 1;
                        } else if (c.kind == 2) {
                            zoneDiffModified += 1;
                        } else if (c.kind == 3) {
                            zoneDiffDeleted += 1;
                        } else if (c.kind == 99) {
                        }

                        zoneDiff.push({
                            msg: c.msg,
                            id: c.id,
                            kind: c.kind,
                        });
                        selectedDiff.push(c.id);
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
    $effect(() => {
        computeDiff(domain, zoneTo, zoneFrom);
    });
</script>

{#if !zoneDiff}
    <div class="my-2 text-center">
        <Spinner color="warning" />
        <p>{$t("wait.exporting")}</p>
    </div>
{:else}
    <DiffZoneView
        {disabled}
        {nodiff}
        {selectable}
        bind:selectedDiff
        {zoneDiff}
        on:change={dispatchSelectionSummary}
    />
{/if}
