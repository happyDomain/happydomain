<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2025 happyDomain
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

    import { Input, InputGroup, InputGroupText } from "@sveltestrap/sveltestrap";

    import { validateDomain } from "$lib/dns";
    import type { Domain } from "$lib/model/domain";
    import { t } from "$lib/translations";

    const dispatch = createEventDispatcher();

    interface Props {
        autofocus: boolean;
        origin: Domain;
        value: string;
    }

    let {
        autofocus = false,
        origin,
        value = $bindable(),
    }: Props = $props();

    function endsWithOrigin(value: string) {
        return value.endsWith(origin.domain);
    }

    let newDomainAppend: string = $derived(endsWithOrigin(value) ? null : (value.length > 0 ? "." + origin.domain : origin.domain));
    let validDomain: boolean | undefined = $derived(value ? (endsWithOrigin(value) ? validateDomain(value.replace(/.$/, ""), origin.domain) : validateDomain(value, origin.domain)) : undefined);

    $effect(() => {
        dispatch("validity-changed", validDomain);
    });
</script>

<InputGroup>
    <Input
        {autofocus}
        class="font-monospace"
        placeholder={$t("domains.placeholder-new-sub")}
        invalid={validDomain === false}
        valid={validDomain === true}
        bind:value
    />
    {#if newDomainAppend}
        <InputGroupText class="font-monospace">
            {newDomainAppend}
        </InputGroupText>
    {/if}
</InputGroup>
