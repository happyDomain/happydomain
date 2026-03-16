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
    import { createEventDispatcher } from "svelte";

    import type { FullCorrection } from "$lib/model/correction";

    const dispatch = createEventDispatcher();

    interface Props {
        disabled?: boolean;
        nodiff?: import("svelte").Snippet;
        selectable?: boolean;
        selectedDiff?: Array<string> | null;
        zoneDiff: Array<FullCorrection>;
    }

    let {
        disabled = false,
        nodiff,
        selectable = false,
        selectedDiff = $bindable(null),
        zoneDiff,
    }: Props = $props();
</script>

{#if zoneDiff.length == 0}
    {#if nodiff}{@render nodiff()}{:else}Aucune différence.{/if}
{:else}
    {#each zoneDiff as line (line.id)}
        <div
            class="font-monospace"
            class:col={selectable}
            class:form-check={selectable}
            class:text-success={line.kind == 1}
            class:text-warning={line.kind == 2}
            class:text-danger={line.kind == 3}
            class:text-info={line.kind == 99}
            style={selectable
                ? "white-space: pre-line;"
                : "white-space: pre-line; text-indent: -1em; margin-left: 1em"}
        >
            {#if selectable}
                <input
                    type="checkbox"
                    class="form-check-input"
                    id="correction-{line.id}"
                    bind:group={selectedDiff}
                    value={line.id}
                    {disabled}
                    onchange={() => dispatch("change")}
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
