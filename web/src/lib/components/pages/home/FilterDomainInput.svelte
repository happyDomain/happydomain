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
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { addDomain } from "$lib/api/domains";
    import { fqdn, validateDomain } from "$lib/dns";
    import { domains, domains_by_name, refreshDomains } from '$lib/stores/domains';
    import { filteredName, filteredProvider } from '$lib/stores/home';
    import { providers } from '$lib/stores/providers';
    import { t } from "$lib/translations";

    interface Props {
        autofocus?: boolean;
        noButton?: boolean;
        [key: string]: any
    }

    let { autofocus = false, noButton = false, ...rest }: Props = $props();

    let addingNewDomain = $state(false);
    async function addDomainToProvider(e: SubmitEvent) {
        e.preventDefault();

        addingNewDomain = true;

        if ($filteredProvider) {
            addDomain($filteredName, $filteredProvider).then(
                (domain) => {
                    addingNewDomain = false;
                    filteredName.set("");
                    refreshDomains();
                },
                (error) => {
                    addingNewDomain = false;
                    throw error;
                },
            );
        } else if ($providers && $providers.length == 1) {
            addDomain($filteredName, $providers[0]).then(
                (domain) => {
                    addingNewDomain = false;
                    filteredName.set("");
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

    let newDomainState: boolean | undefined = $derived(validateNewDomain($filteredName));
    function validateNewDomain(val: string): boolean | undefined {
        return validateDomain(val, "", false);
    }
</script>

<form onsubmit={addDomainToProvider}>
    <ListGroup {...rest}>
        <ListGroupItem class="d-flex justify-content-between align-items-center p-0">
            <InputGroup>
                <label
                    for="newdomaininput"
                    class="ms-2 my-1 text-center text-muted"
                    style="font-size: 1.6rem"
                >
                    {#if ($domains && $filteredName && $domains.filter((dn) => dn.domain == fqdn($filteredName, "")).length == 0) || ($domains && $domains.length == 0)}
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
                    disabled={addingNewDomain}
                    placeholder={$t("domains.placeholder-search")}
                    invalid={$filteredName.length
                        ? newDomainState !== undefined && !newDomainState
                        : undefined}
                    valid={$filteredName.length ? newDomainState : undefined}
                    style="border:none;box-shadow:none;z-index:0"
                    bind:value={$filteredName}
                />
                {#if !noButton && $filteredName.length && (!$domains_by_name[fqdn($filteredName, "")] || !$filteredProvider || !$domains_by_name[fqdn($filteredName, "")].reduce((acc, d) => acc || d.id_provider == $filteredProvider._id, false))}
                    <Button type="submit" outline color="primary" disabled={addingNewDomain}>
                        {#if addingNewDomain}
                            <Spinner size="sm" class="me-1" />
                        {/if}
                        {$t("common.add-new-thing", { thing: $t("domains.kind") })}
                    </Button>
                {/if}
            </InputGroup>
        </ListGroupItem>
    </ListGroup>
</form>
