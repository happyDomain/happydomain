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
 import { goto } from '$app/navigation';

 import {
     Button,
     Icon,
     Input,
     InputGroup,
     ListGroup,
     ListGroupItem,
     Spinner,
 } from '@sveltestrap/sveltestrap';

 import { addDomain } from '$lib/api/domains';
 import { validateDomain } from '$lib/dns';
 import type { Provider } from '$lib/model/provider';
 import { refreshDomains } from '$lib/stores/domains';
 import { toasts } from '$lib/stores/toasts';
 import { t } from '$lib/translations';

 export let autofocus = false;
 export let provider: Provider|null = null;
 export let value = "";

 let addingNewDomain = false;
 let newDomainState: boolean|undefined = undefined;

 function addDomainToProvider() {
     addingNewDomain = true;

     if (!provider) {
         goto('/domains/new/' + encodeURIComponent(value));
     } else {
         addDomain(value, provider)
         .then(
             (domain) => {
                 addingNewDomain = false;
                 toasts.addToast({
                     title: $t('domains.attached-new'),
                     message: $t('domains.added-success', { domain: domain.domain }),
                     href: '/domains/' + domain.domain,
                     color: 'success',
                     timeout: 5000,
                 });

                 value = "";
                 refreshDomains();
             },
             (error) => {
                 addingNewDomain = false;
                 throw error;
             }
         );
     }
 }

 function validateNewDomain(val: string|undefined): boolean|undefined {
     if (val) {
         newDomainState = validateDomain(val);
     } else {
         newDomainState = validateDomain(value);
     }

     return newDomainState;
 }

 function inputChange(event: Event) {
     if (event instanceof InputEvent) {
         validateNewDomain(event.data?value+event.data:value.substring(0,value.length-1));
     }
 }
</script>

<ListGroup {...$$restProps}>
    <form on:submit|preventDefault={addDomainToProvider}>
        <ListGroupItem class="d-flex justify-content-between align-items-center">
            <InputGroup>
                <label for="newdomaininput" class="text-center" style="width: 50px; font-size: 2.3rem">
                    <Icon name="plus" />
                </label>
                <Input
                    id="newdomaininput"
                    {autofocus}
                    class="font-monospace"
                    placeholder={$t('domains.placeholder-new')}
                    invalid={value.length ? newDomainState !== undefined && !newDomainState : undefined}
                    valid={value.length ? newDomainState : undefined}
                    style="border:none;box-shadow:none;z-index:0"
                    bind:value={value}
                    on:input={inputChange}
                />
                {#if value.length}
                    <Button
                        type="submit"
                        outline
                        color="primary"
                        disabled={addingNewDomain}
                    >
                        {#if addingNewDomain}
                            <Spinner size="sm" class="me-1" />
                        {/if}
                        {$t('common.add-new-thing', { thing: $t('domains.kind') })}
                    </Button>
                {/if}
            </InputGroup>
        </ListGroupItem>
    </form>
</ListGroup>
