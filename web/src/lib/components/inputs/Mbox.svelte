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
    import { Input, InputGroup, InputGroupText } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import { emailToMbox, mboxToEmail } from "$lib/utils/mbox";

    interface Props {
        id: string;
        // Origin of the zone, the mailbox being relative to it.
        origin: string;
        placeholder?: string;
        readonly?: boolean;
        required?: boolean;
        // Mailbox in its DNS form (john\.doe, hostmaster.other.org.).
        value: string | undefined;
    }

    let {
        id,
        origin,
        placeholder,
        readonly = false,
        required = false,
        value = $bindable(),
    }: Props = $props();

    let zone = $derived(origin.replace(/\.$/, ""));

    function display(mbox: string | undefined): string {
        if (!mbox || mbox === "@") return "";
        // An address typed as is, before this input existed: shown untouched.
        if (mbox.includes("@")) return mbox;
        const email = mboxToEmail(mbox, origin);
        if (!email) return "";
        // Only a mailbox stored relative to the zone itself is shown without
        // its domain: one typed with its domain keeps it.
        return !mbox.endsWith(".") && email.domain === zone.toLowerCase()
            ? email.local
            : email.local + "@" + email.domain;
    }

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

    let mbox: string | null = $derived(emailToMbox(text, origin));

    let feedback: string | undefined = $derived(
        text.trim().length > 0 && mbox === null ? $t("errors.mbox") : undefined,
    );

    // Until an "@" is typed, the address is in the zone's domain.
    let suffix: string | null = $derived(zone !== "" && !text.includes("@") ? "@" + zone : null);

    function oninput() {
        if (mbox !== null) {
            emitted = mbox;
            value = mbox;
        }
    }

    function normalize() {
        if (mbox !== null) text = display(mbox);
    }
</script>

<InputGroup size="sm">
    <Input
        {id}
        type="text"
        inputmode="email"
        {feedback}
        invalid={feedback !== undefined}
        {placeholder}
        plaintext={readonly}
        {readonly}
        {required}
        autocomplete="off"
        spellcheck="false"
        bind:value={text}
        on:input={oninput}
        on:blur={normalize}
    />
    {#if suffix !== null}
        <InputGroupText>{suffix}</InputGroupText>
    {/if}
</InputGroup>
