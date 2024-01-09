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

<script context="module" lang="ts">
 import type { ModalController } from '$lib/model/modal_controller';

 export const controls: ModalController = { };
</script>

<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 // @ts-ignore
 import { escape } from 'html-escaper';

 import {
     Button,
     Icon,
     Input,
     InputGroup,
     InputGroupText,
     Modal,
     ModalBody,
     ModalFooter,
     ModalHeader,
     Spinner,
 } from '@sveltestrap/sveltestrap';

 import { addZoneService } from '$lib/api/zone';
 import { fqdn, validateDomain } from '$lib/dns';
 import type { Domain, DomainInList } from '$lib/model/domain';
 import type { Zone } from '$lib/model/zone';
 import { t } from '$lib/translations';

 const dispatch = createEventDispatcher();

 export let isOpen = false;
 const toggle = () => (isOpen = !isOpen);

 $: if (isOpen) {
     value = "";
 }

 export let dn: string;
 export let origin: Domain | DomainInList;
 export let value: string = "";
 export let zone: Zone;

 let newDomainState: boolean | undefined = undefined;
 $: newDomainState = value?validateNewSubdomain(value):undefined;

 let endsWithOrigin = false;
 $: endsWithOrigin = value.length > origin.domain.length && (
     value.substring(value.length - origin.domain.length) === origin.domain ||
     value.substring(value.length - origin.domain.length + 1) === origin.domain.substring(0, origin.domain.length - 1)
 )

 let newDomainAppend: string | null = null;
 $: {
     if (endsWithOrigin) {
         newDomainAppend = null;
     } else if (value.length > 0) {
         newDomainAppend = '.' + origin.domain;
     } else {
         newDomainAppend = origin.domain;
     }
 }

 function validateNewSubdomain(value: string): boolean | undefined {
     // Check domain is valid
     newDomainState = validateDomain(
         value,
         (value.length > origin.domain.length && value.substring(value.length - origin.domain.length) === origin.domain)?origin.domain:""
     );

     // Check domain doesn't already exists
     if (zone.services[value]) {
         return false;
     } else if (value.length > origin.domain.length && value.indexOf(origin.domain) == value.length - origin.domain.length && zone.services[value.substring(0, value.length - origin.domain.length)]) {
         return false;
     } else if (value.length > origin.domain.length && value.indexOf(origin.domain.substring(0, origin.domain.length - 1)) == value.length - origin.domain.length + 1 && zone.services[value.substring(0, value.length - origin.domain.length)]) {
         return false;
     }

     return newDomainState;
 }

 let addAliasInProgress = false;
 function submitAliasForm() {
     if (validateNewSubdomain(value)) {
         addAliasInProgress = true;
         addZoneService(origin, zone.id, {_domain: value, _svctype: "svcs.CNAME", Service: {Target: dn?dn:"@"}}).then(
             (z) => {
                 dispatch("update-zone-services", z);
                 addAliasInProgress = false;
                 toggle();
             },
             (err) => {
                 addAliasInProgress = false;
                 throw err;
             }
         );
     }
 }

 function Open(domain: string): void {
     dn = domain;
     isOpen = true;
 }

 controls.Open = Open;
</script>

<Modal
    {isOpen}
    {toggle}
>
    <ModalHeader>
        {$t('domains.add-an-alias')} {origin.domain}
    </ModalHeader>
    <ModalBody>
        <form
            id="addAliasForm"
            on:submit|preventDefault={submitAliasForm}
        >
            <p>
                {@html $t('domains.alias-creation', {"domain": `<span class="font-monospace">${escape(fqdn(dn, origin.domain))}</span>`})}
                <InputGroup>
                    <Input
                        autofocus
                        class="font-monospace"
                        placeholder={$t('domains.placeholder-new-sub')}
                        invalid={newDomainState === false}
                        valid={newDomainState === true}
                        bind:value={value}
                    />
                    {#if newDomainAppend}
                        <InputGroupText class="font-monospace">
                            {newDomainAppend}
                        </InputGroupText>
                    {/if}
                </InputGroup>
            </p>
            {#if newDomainState}
                <div class="mt-3 text-center">
                    {$t('domains.alias-creation-sample')}<br>
                    {#if endsWithOrigin}
                        <span class="font-monospace text-no-wrap">{fqdn(value, "")}</span>
                    {:else}
                        <span class="font-monospace text-no-wrap">{fqdn(value, origin.domain)}</span>
                    {/if}
                    <Icon class="mx-1" name="arrow-right" />
                    <span class="font-monospace text-no-wrap">{fqdn(dn, origin.domain)}</span>
                </div>
            {/if}
        </form>
    </ModalBody>
    <ModalFooter>
        <Button
            color="secondary"
            outline
            on:click={toggle}
        >
            {$t('common.cancel')}
        </Button>
        <Button
            type="submit"
            disabled={newDomainState !== true || addAliasInProgress}
            form="addAliasForm"
            color="primary"
        >
            {#if addAliasInProgress}
                <Spinner size="sm" />
            {/if}
            {$t('domains.add-alias')}
        </Button>
    </ModalFooter>
</Modal>
