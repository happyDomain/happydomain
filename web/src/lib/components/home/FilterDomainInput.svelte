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
    import { goto } from "$app/navigation";

    import {
        Button,
        Icon,
        Input,
        InputGroup,
        ListGroup,
        ListGroupItem,
    } from "@sveltestrap/sveltestrap";

    import { addDomain } from "$lib/api/domains";
    import { fqdn, validateDomain } from "$lib/dns";
    import { domains, refreshDomains } from '$lib/stores/domains';
    import { filteredName, filteredProvider } from '$lib/stores/home';
    import { providers } from '$lib/stores/providers';
    import { t } from "$lib/translations";

    export let autofocus = false;
    export let noButton = false;

    let addingNewDomain = false;
    async function addDomainToProvider() {
        addingNewDomain = true;

        if ($filteredProvider) {
            addDomain($filteredName, $filteredProvider).then(
                (domain) => {
                    addingNewDomain = false;
                    refreshDomains();
                },
                (error) => {
                    addingNewDomain = false;
                    throw error;
                },
            );
        } else if ($providers.length == 1) {
            addDomain($filteredName, $providers[0]).then(
                (domain) => {
                    addingNewDomain = false;
                    refreshDomains();
                },
                (error) => {
                    addingNewDomain = false;
                    throw error;
                },
            );
        } else {
            goto("/domains/new/" + encodeURIComponent($filteredName));
        }
    }

    let newDomainState: boolean | undefined = undefined;
    function validateNewDomain(val: string | undefined): boolean | undefined {
        if (val) {
            newDomainState = validateDomain(val);
        } else {
            newDomainState = validateDomain($filteredName);
        }

        return newDomainState;
    }

    function inputChange(event: Event) {
        if (event instanceof InputEvent) {
            validateNewDomain(
                event.data ? $filteredName + event.data : $filteredName.substring(0, $filteredName.length - 1),
            );
        }
    }
</script>

<form on:submit|preventDefault={addDomainToProvider}>
    <ListGroup {...$$restProps}>
        <ListGroupItem class="d-flex justify-content-between align-items-center p-0">
            <InputGroup>
                <label
                    for="newdomaininput"
                    class="ms-2 my-1 text-center text-muted"
                    style="font-size: 1.6rem"
                >
                    {#if ($filteredName && $domains.filter((dn) => dn.domain == fqdn($filteredName, "")).length == 0) || ($domains && $domains.length == 0)}
                        <Icon name="plus-lg" />
                    {:else}
                        <Icon name="search" />
                    {/if}
                </label>
                <Input
                    id="newdomaininput"
                    {autofocus}
                    autocomplete="off"
                    class="font-monospace"
                    placeholder={$t("domains.placeholder-search")}
                    invalid={$filteredName.length
                        ? newDomainState !== undefined && !newDomainState
                        : undefined}
                    valid={$filteredName.length ? newDomainState : undefined}
                    style="border:none;box-shadow:none;z-index:0"
                    bind:value={$filteredName}
                    on:input={inputChange}
                />
                {#if !noButton && $filteredName.length && $domains.filter((dn) => dn.domain == fqdn($filteredName, "")).length == 0}
                    <Button type="submit" outline color="primary">
                        {$t("common.add-new-thing", { thing: $t("domains.kind") })}
                    </Button>
                {/if}
            </InputGroup>
        </ListGroupItem>
    </ListGroup>
</form>
