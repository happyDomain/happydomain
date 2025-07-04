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

    import { Button, Collapse, FormGroup, Input, Spinner } from "@sveltestrap/sveltestrap";

    import SelectType from "./SelectType.svelte";
    import SelectResolver from "./SelectResolver.svelte";
    import type { Domain } from "$lib/model/domain";
    import type { ResolverForm } from "$lib/model/resolver";
    import { t } from "$lib/translations";

    const dispatch = createEventDispatcher();

    export let value: ResolverForm = { domain: "", type: "ANY", resolver: "local" };
    export let showDNSSEC = false;

    export let sortedDomains: Array<Domain> = [];
    export let request_pending = false;

    function submitRequest(): void {
        request_pending = true;
        dispatch("submit", { value, showDNSSEC });
    }
</script>

<form class="pt-3 pb-5" on:submit|preventDefault={submitRequest}>
    <FormGroup>
        <label for="domain">
            {$t("common.domain")}
        </label>
        <Input
            aria-describedby="domainHelpBlock"
            id="domain"
            class="font-monospace"
            list="my-domains"
            required
            placeholder="happydomain.org"
            bind:value={value.domain}
        />
        <div id="domainHelpBlock" class="form-text">
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
    </FormGroup>

    <div class="text-center mb-3">
        <Button type="button" color="secondary" id="settingsToggler">
            {$t("resolver.advanced")}
        </Button>
    </div>

    <Collapse toggler="#settingsToggler">
        <FormGroup>
            <label for="select-type">
                {$t("common.field")}
            </label>
            <SelectType
                aria-describedby="typeHelpBlock"
                id="select-type"
                required
                bind:value={value.type}
            />
            <div id="typeHelpBlock" class="form-text">
                {$t("resolver.field-description")}
            </div>
        </FormGroup>

        <FormGroup>
            <label for="select-resolver">
                {$t("common.resolver")}
            </label>
            <SelectResolver
                aria-describedby="resolverHelpBlock"
                id="select-resolver"
                required
                bind:value={value.resolver}
            />
            <div id="resolverHelpBlock" class="form-text">
                {$t("resolver.resolver-description")}
            </div>
        </FormGroup>

        {#if value.resolver === "custom"}
            <FormGroup>
                <label for="custom-resolver">
                    {$t("resolver.custom")}
                </label>
                <Input
                    aria-describedby="customResolverHelpBlock"
                    id="custom-resolver"
                    required={value.resolver === "custom"}
                    placeholder="127.0.0.1"
                    bind:value={value.custom}
                />
                <div id="customResolverHelpBlock" class="form-text">
                    {$t("resolver.custom-description")}
                </div>
            </FormGroup>
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

    <div class="mx-3">
        <Button type="submit" class="float-end" color="primary" disabled={request_pending}>
            {#if request_pending}
                <Spinner size="sm" />
            {/if}
            {$t("common.run")}
        </Button>
    </div>
</form>
