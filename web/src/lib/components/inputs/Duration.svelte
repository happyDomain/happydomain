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
    import { Input, InputGroupText } from "@sveltestrap/sveltestrap";

    import { locale, t } from "$lib/translations";
    import {
        formatBindDuration,
        formatHumanDuration,
        parseBindDuration,
    } from "$lib/utils/duration";

    interface Props {
        id: string;
        max?: number;
        placeholder?: string;
        readonly?: boolean;
        plaintext?: boolean;
        required?: boolean;
        // Seconds. A string is an older shorthand ("1m") still read as such.
        value: number | string | undefined;
        onfocus?: () => void;
        onblur?: () => void;
    }

    let {
        id,
        max,
        placeholder,
        readonly = false,
        plaintext = false,
        required = false,
        value = $bindable(),
        onfocus,
        onblur,
    }: Props = $props();

    function toSeconds(v: unknown): number | null {
        if (typeof v === "number") return v;
        if (typeof v === "string") return parseBindDuration(v);
        return null;
    }

    function display(v: unknown): string {
        const seconds = toSeconds(v);
        return seconds === null ? (typeof v === "string" ? v : "") : formatBindDuration(seconds);
    }

    // What the user typed is kept as is while they type: rewriting "1h3" into
    // "1h3s" under their fingers would get in the way.
    // svelte-ignore state_referenced_locally
    let text = $state(display(value));
    let emitted: unknown = value;

    // The value changed from elsewhere (another field, a reset): show it.
    $effect(() => {
        if (value !== emitted) {
            emitted = value;
            text = display(value);
        }
    });

    let seconds: number | null = $derived(parseBindDuration(text));

    let feedback: string | undefined = $derived(
        text.trim().length === 0
            ? undefined
            : seconds === null
              ? $t("errors.duration")
              : max !== undefined && seconds > max
                ? $t("errors.too-high", { max })
                : undefined,
    );

    let hint: string | null = $derived(
        seconds !== null && feedback === undefined ? formatHumanDuration(seconds, $locale) : null,
    );

    function oninput() {
        if (seconds !== null && feedback === undefined) {
            emitted = seconds;
            value = seconds;
        }
    }

    function normalize() {
        if (seconds !== null) text = formatBindDuration(seconds);
        onblur?.();
    }
</script>

<Input
    {id}
    type="text"
    {feedback}
    invalid={feedback !== undefined}
    {placeholder}
    {plaintext}
    readonly={plaintext || readonly}
    {required}
    autocomplete="off"
    spellcheck="false"
    bind:value={text}
    on:input={oninput}
    on:focus={() => onfocus?.()}
    on:blur={normalize}
/>
{#if hint !== null}
    <InputGroupText>= {hint}</InputGroupText>
{/if}
