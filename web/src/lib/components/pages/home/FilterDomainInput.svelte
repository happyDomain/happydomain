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
    import { domainLinks } from "$links";
    import { goto } from "$app/navigation";
    import { resolve } from "$app/paths";
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
    import DomainNameIssue from "$lib/components/inputs/DomainNameIssue.svelte";
    import { controls as newDomainControls } from "$lib/components/modals/NewDomain.svelte";
    import { domainNameIssue, domainNameIssueBlocks, fqdn, validateDomain } from "$lib/dns";
    import { domains, refreshDomains } from "$lib/stores/domains";
    import { filterDomains, filteredName, filteredProvider } from "$lib/stores/home";
    import type { Provider } from "$lib/model/provider";
    import { providers } from "$lib/stores/providers";
    import { t } from "$lib/translations";

    interface Props {
        autofocus?: boolean;
        noButton?: boolean;
        [key: string]: unknown;
    }

    let { autofocus = false, noButton = false, ...rest }: Props = $props();

    let addingNewDomain = $state(false);
    async function submitFilterDomainInput(e: SubmitEvent) {
        e.preventDefault();

        if (actionAddDomain) {
            if (!newDomainBlocked) addDomainToProvider();
        } else if (matchingDomains.length > 0) {
            goto(domainLinks().zone(encodeURIComponent(matchingDomains[0].id)));
        }
    }

    function addDomainToProvider() {
        if ($filteredProvider) {
            addDomainAndOpen($filteredProvider);
        } else if ($providers && $providers.length == 1) {
            addDomainAndOpen($providers[0]);
        } else {
            newDomainControls.Open({ ondone: addDomainAndOpen });
        }
    }

    // addDomainAndOpen takes the user to the domain once added. The domain
    // page looks the domain up in the list: it has to be refreshed first.
    function addDomainAndOpen(provider: Provider) {
        addingNewDomain = true;
        addDomain($filteredName, provider).then(
            async (domain) => {
                await refreshDomains();
                addingNewDomain = false;
                filteredName.set("");
                goto(domainLinks().zone(domain.domain));
            },
            (error) => {
                addingNewDomain = false;
                throw error;
            },
        );
    }

    let newDomainState: boolean | undefined = $derived(validateNewDomain($filteredName));
    function validateNewDomain(val: string): boolean | undefined {
        return validateDomain(val, "", false);
    }
    let newDomainIssue = $derived(domainNameIssue($filteredName));
    let newDomainBlocked = $derived(domainNameIssueBlocks(newDomainIssue));

    let filteredDomains = $derived(filterDomains($domains, $filteredName, $filteredProvider, null));
    let matchingDomains = $derived(
        filteredDomains.filter((dn) => dn.domain == fqdn($filteredName, "")),
    );
    let actionAddDomain = $derived($filteredName && matchingDomains.length == 0);
</script>

<form onsubmit={submitFilterDomainInput} {...rest}>
    <ListGroup>
        <ListGroupItem class="d-flex justify-content-between align-items-center p-0">
            <InputGroup>
                <label
                    for="newdomaininput"
                    class="ms-2 my-1 text-center text-muted"
                    style="font-size: 1.6rem"
                >
                    {#if actionAddDomain}
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
                {#if !noButton && $filteredName.length}
                    <Button
                        type="submit"
                        outline
                        color={actionAddDomain ? "primary" : "secondary"}
                        disabled={addingNewDomain || newDomainBlocked}
                    >
                        {#if addingNewDomain}
                            <Spinner size="sm" class="me-1" />
                        {/if}
                        {#if actionAddDomain}
                            {$t("common.add-new-thing", { thing: $t("domains.kind") })}
                        {:else}
                            {$t("domains.view.title")}
                        {/if}
                    </Button>
                {/if}
            </InputGroup>
        </ListGroupItem>
    </ListGroup>
    {#if newDomainIssue}
        <DomainNameIssue class="ms-2" issue={newDomainIssue} />
    {/if}
</form>
