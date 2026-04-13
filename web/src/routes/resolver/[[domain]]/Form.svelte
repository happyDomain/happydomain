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
    import { preventDefault } from 'svelte/legacy';

    import { createEventDispatcher } from "svelte";

    import { Button, Collapse, FormGroup, Input, Spinner } from "@sveltestrap/sveltestrap";

    import SelectType from "./SelectType.svelte";
    import SelectResolver from "./SelectResolver.svelte";
    import type { Domain } from "$lib/model/domain";
    import type { ResolverForm } from "$lib/model/resolver";
    import { t } from "$lib/translations";

    const dispatch = createEventDispatcher();

    interface Props {
        class?: string;
        value?: ResolverForm;
        showDNSSEC?: boolean;
        sortedDomains?: Array<Domain>;
        request_pending?: boolean;
    }

    let {
        class: className = "",
        value = $bindable({ domain: "", type: "ANY", resolver: "local" }),
        showDNSSEC = $bindable(false),
        sortedDomains = [],
        request_pending = $bindable(false)
    }: Props = $props();

    function submitRequest(): void {
        request_pending = true;
        dispatch("submit", { value: $state.snapshot(value), showDNSSEC: $state.snapshot(showDNSSEC) });
    }
</script>

<form class={className} onsubmit={preventDefault(submitRequest)}>
    <FormGroup floating label={$t("common.domain")}>
        <Input
            aria-describedby="domainHelpBlock"
            id="domain"
            class="font-monospace"
            list="my-domains"
            required
            placeholder={$t("common.domain")}
            bind:value={value.domain}
        />
    </FormGroup>
    <div id="domainHelpBlock" class="form-text mb-3 mt-n2">
        {@html $t("resolver.domain-description", {
            domain: `<a href="/resolver/wikipedia.org" class="font-monospace">wikipedia.org</a>`,
        })}
    </div>
    <datalist id="my-domains">
        {#each sortedDomains as dn (dn.id)}
            <option>
                {dn.domain}
            </option>
        {/each}
    </datalist>

    <div class="mb-3">
        <button
            type="button"
            class="btn btn-sm btn-link text-body-secondary text-decoration-none p-0 d-inline-flex align-items-center gap-1"
            id="settingsToggler"
        >
            <i class="bi bi-sliders"></i>
            {$t("resolver.advanced")}
        </button>
    </div>

    <Collapse toggler="#settingsToggler">
        <FormGroup floating label={$t("common.field")}>
            <SelectType
                aria-describedby="typeHelpBlock"
                id="select-type"
                required
                bind:value={value.type}
            />
        </FormGroup>
        <div id="typeHelpBlock" class="form-text mb-3 mt-n2">
            {$t("resolver.field-description")}
        </div>

        <FormGroup floating label={$t("common.resolver")}>
            <SelectResolver
                aria-describedby="resolverHelpBlock"
                id="select-resolver"
                required
                bind:value={value.resolver}
            />
        </FormGroup>
        <div id="resolverHelpBlock" class="form-text mb-3 mt-n2">
            {$t("resolver.resolver-description")}
        </div>

        {#if value.resolver === "custom"}
            <FormGroup floating label={$t("resolver.custom")}>
                <Input
                    aria-describedby="customResolverHelpBlock"
                    id="custom-resolver"
                    required={value.resolver === "custom"}
                    placeholder={$t("resolver.custom")}
                    bind:value={value.custom}
                />
            </FormGroup>
            <div id="customResolverHelpBlock" class="form-text mb-3 mt-n2">
                {$t("resolver.custom-description")}
            </div>
        {/if}

        <Input
            type="checkbox"
            label={$t("resolver.showDNSSEC")}
            id="showDNSSEC"
            bind:value={showDNSSEC}
            name="showDNSSEC"
            class="mb-3"
        />
    </Collapse>

    <div class="d-grid">
        <Button type="submit" color="primary" disabled={request_pending}>
            {#if request_pending}
                <Spinner size="sm" class="me-1" />
            {/if}
            <i class="bi bi-search me-1"></i>
            {$t("common.run")}
        </Button>
    </div>
</form>
