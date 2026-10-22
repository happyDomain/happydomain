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
    import { t } from "$lib/translations";

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
    {#if nodiff}{@render nodiff()}{:else}{$t("domains.apply.nodiff")}{/if}
{:else}
    <div class="diff-lines">
        {#each zoneDiff as line (line.id)}
            {@const kindClasses = {
                "hd-diff-add": line.kind == 1,
                "hd-diff-mod": line.kind == 2,
                "hd-diff-del": line.kind == 3,
                "hd-diff-info": line.kind == 99,
            }}
            <div class="diff-line">
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
                        class={["diff-msg font-monospace", kindClasses]}
                        class:unselected={!selectedDiff?.includes(line.id)}
                        for="correction-{line.id}">{line.msg}</label
                    >
                {:else}
                    <span class={["diff-msg font-monospace", kindClasses]}>{line.msg}</span>
                {/if}
            </div>
        {/each}
    </div>
{/if}

<style>
    .diff-lines {
        display: flex;
        flex-direction: column;
        gap: 2px;
    }

    .diff-line {
        display: flex;
        align-items: flex-start;
        gap: 0.5rem;
    }

    /* Outside the coloured block: Bootstrap's 0.25em top margin centres the
       1em box on a 1.5 line-height row, plus the block's top padding. */
    .diff-line .form-check-input {
        flex-shrink: 0;
        margin: calc(0.25rem + 0.25em) 0 0;
    }

    /* Hanging indent: wrapped lines of a long record fall under the text,
       not under the leading +/-/± marker. */
    .diff-msg {
        flex: 1;
        min-width: 0;
        padding: 0.25rem 0.5rem 0.25rem calc(0.6rem + 1em);
        text-indent: -1em;
        white-space: pre-line;
        overflow-wrap: anywhere;
        transition: background-color 0.15s;
    }

    /* A change left out of the apply keeps its line and colour, but loses
       its fill. */
    .diff-msg.unselected {
        background: transparent;
    }

    label.diff-msg {
        cursor: pointer;
    }
</style>
