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
 import {
     Modal,
     ModalBody,
     ModalFooter,
     ModalHeader,
 } from '@sveltestrap/sveltestrap';

 import SettingsStateButtons from '$lib/components/providers/SettingsStateButtons.svelte';
 import CustomForm from '$lib/components/CustomForm.svelte';
 import ProviderSelector from '$lib/components/providers/Selector.svelte';
 import ResourceInput from '$lib/components/resources/basic.svelte';
 import { ProviderForm } from '$lib/model/provider_form';
 import { providersSpecs, refreshProviders } from '$lib/stores/providers';
 import { t } from '$lib/translations';

 export let isOpen = false;

 let form = new ProviderForm("", () => {
     isOpen = false;
     refreshProviders();
 });

 function previous() {
     if (form.state < 0) {
         isOpen = false;
     } else {
         form.previousState();
         form = form;
     }
 }

 function selectProvider(event: CustomEvent<{ptype: string}>) {
     form.ptype = event.detail.ptype;
     form.changeState(0).then((res) => form.form = res);
 }

 function toggle(): void {
     isOpen = !isOpen;
 }
</script>

<Modal
    {isOpen}
    scrollable
    size="lg"
    {toggle}
>
    <ModalHeader {toggle}>
        {$t('provider.new-form')}
    </ModalHeader>
    <ModalBody>
        {#if form.state < 0}
            <p>
                {$t('provider.select-provider')}
            </p>
            <ProviderSelector on:provider-selected={selectProvider} />
        {:else}
            <form
                id="providermodal"
                on:submit|preventDefault={() => form.nextState()}
            >
                {#if form.form}
                    <CustomForm
                        form={form.form}
                        bind:value={form.value.Provider}
                        on:input={(event) => form.value.Provider = event.detail}
                    >
                        {#if form.state === 0}
                            <ResourceInput
                                id="src-name"
                                edit
                                index="0"
                                specs={{label: $t('provider.name-your'), description: $t('domains.give-explicit-name'), placeholder: $providersSpecs?($providersSpecs[form.ptype].name + ' account 1'):undefined, required: true}}
                                bind:value={form.value._comment}
                                on:input={(event) => form.value._comment = event.detail}
                            />
                        {/if}
                    </CustomForm>
                {/if}
            </form>
        {/if}
    </ModalBody>
    <ModalFooter>
        <SettingsStateButtons
            class="d-flex justify-content-end"
            submitForm="providermodal"
            form={form.form}
            nextInProgress={form.nextInProgress}
            previousInProgress={form.previousInProgress}
            on:previous-state={previous}
        />
    </ModalFooter>
</Modal>
