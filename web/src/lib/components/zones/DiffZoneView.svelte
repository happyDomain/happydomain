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

    // Any kind but addition, modification and deletion is informational.
    function kindName(kind: number | undefined): "add" | "mod" | "del" | "info" {
        switch (kind) {
            case 1:
                return "add";
            case 2:
                return "mod";
            case 3:
                return "del";
            default:
                return "info";
        }
    }
</script>

{#if zoneDiff.length == 0}
    {#if nodiff}{@render nodiff()}{:else}{$t("domains.apply.nodiff")}{/if}
{:else}
    <div class="d-flex flex-column gap-1">
        {#each zoneDiff as line (line.id)}
            {@const kind = kindName(line.kind)}
            {@const kindClasses = `hd-diff-${kind}`}
            <div class="diff-line diff-line-{kind} d-flex align-items-start gap-2">
                {#if selectable}
                    <input
                        type="checkbox"
                        class="form-check-input flex-shrink-0"
                        id="correction-{line.id}"
                        bind:group={selectedDiff}
                        value={line.id}
                        {disabled}
                        onchange={() => dispatch("change")}
                    />
                    <label
                        class={["diff-msg flex-fill min-w-0 font-monospace", kindClasses]}
                        class:bg-transparent={!selectedDiff?.includes(line.id)}
                        for="correction-{line.id}">{line.msg}</label
                    >
                {:else}
                    <span class={["diff-msg flex-fill min-w-0 font-monospace", kindClasses]}
                        >{line.msg}</span
                    >
                {/if}
            </div>
        {/each}
    </div>
{/if}

<style>
    /* Outside the coloured block: Bootstrap's 0.25em top margin centres the
       1em box on a 1.5 line-height row, plus the block's top padding. */
    .diff-line .form-check-input {
        margin: calc(0.25rem + 0.25em) 0 0;
        border-color: var(--diff-check);
    }

    /* The checkbox takes the colour of the change it selects. */
    .diff-line-add {
        --diff-check: var(--hd-diff-add-line);
    }
    .diff-line-mod {
        --diff-check: var(--hd-diff-mod-line);
    }
    .diff-line-del {
        --diff-check: var(--hd-diff-del-line);
    }
    .diff-line-info {
        --diff-check: var(--hd-diff-info-line);
    }

    .diff-line .form-check-input:checked {
        background-color: var(--diff-check);
    }

    .diff-line .form-check-input:focus {
        box-shadow: 0 0 0 0.25rem color-mix(in srgb, var(--diff-check) 25%, transparent);
    }

    /* Hanging indent: wrapped lines of a long record fall under the text,
       not under the leading +/-/± marker. */
    .diff-msg {
        padding: 0.25rem 0.5rem 0.25rem calc(0.6rem + 1em);
        text-indent: -1em;
        white-space: pre-line;
        overflow-wrap: anywhere;
        transition: background-color 0.15s;
    }

    label.diff-msg {
        cursor: pointer;
    }
</style>
