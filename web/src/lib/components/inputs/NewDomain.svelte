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
    import { createEventDispatcher } from "svelte";

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
    import { validateDomain } from "$lib/dns";
    import type { Provider } from "$lib/model/provider";
    import { refreshDomains } from "$lib/stores/domains";
    import { t } from "$lib/translations";

    const dispatch = createEventDispatcher();

    interface Props {
        addingNewDomain?: boolean;
        autofocus?: boolean;
        noButton?: boolean;
        preAddFunc?: null | ((arg0: string) => Promise<boolean>);
        provider?: Provider | null;
        value?: string;
        [key: string]: any
    }

    let {
        addingNewDomain = $bindable(false),
        autofocus = false,
        noButton = false,
        preAddFunc = null,
        provider = null,
        value = $bindable(""),
        ...rest
    }: Props = $props();

    let formId = "new-domain-form";

    async function addDomainToProvider(e: SubmitEvent) {
        e.preventDefault();

        addingNewDomain = true;

        if (preAddFunc && !(await preAddFunc(value))) {
            addingNewDomain = false;
            return;
        }

        if (!provider) {
            goto("/domains/new/" + encodeURIComponent(value));
        } else {
            addDomain(value, provider).then(
                (domain) => {
                    addingNewDomain = false;
                    value = "";
                    refreshDomains();
                    dispatch("newDomainAdded", domain);
                },
                (error) => {
                    addingNewDomain = false;
                    throw error;
                },
            );
        }
    }

    let newDomainState: boolean | undefined = $derived(validateNewDomain(value));
    function validateNewDomain(val: string): boolean | undefined {
        return validateDomain(val, "", false);
    }
</script>

<form id={formId} onsubmit={addDomainToProvider}>
    <ListGroup {...rest}>
        <ListGroupItem class="d-flex justify-content-between align-items-center p-0">
            <InputGroup>
                <label
                    for="newdomaininput"
                    class="text-center"
                    style="width: 50px; font-size: 2.3rem"
                >
                    <Icon name="plus" />
                </label>
                <Input
                    id="newdomaininput"
                    {autofocus}
                    class="font-monospace"
                    placeholder={$t("domains.placeholder-new")}
                    invalid={value.length
                        ? newDomainState !== undefined && !newDomainState
                        : undefined}
                    valid={value.length ? newDomainState : undefined}
                    style="border:none;box-shadow:none;z-index:0"
                    bind:value
                />
                {#if !noButton && value.length}
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
