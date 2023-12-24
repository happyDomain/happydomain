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
     Col,
     Container,
     Row,
     Spinner,
 } from 'sveltestrap';

 import ImgProvider from '$lib/components/providers/ImgProvider.svelte';
 import SettingsStateButtons from '$lib/components/providers/SettingsStateButtons.svelte';
 import CustomForm from '$lib/components/CustomForm.svelte';
 import ResourceInput from '$lib/components/resources/basic.svelte';
 import { ProviderForm } from '$lib/model/provider_form';
 import type { ProviderSettingsState } from '$lib/model/provider_settings';
 import { providersSpecs, refreshProviders, refreshProvidersSpecs } from '$lib/stores/providers';
 import { t } from '$lib/translations';

 // Load required data
 if ($providersSpecs == null) refreshProvidersSpecs();

 // Inputs
 export let edit = false;
 export let ptype: string;
 export let state: number;
 export let providerId: string | null = null;
 export let value: ProviderSettingsState | null = null;

 //
 let form = new ProviderForm(ptype, () => refreshProviders().then(() => goto('/')), providerId, value, () => {
     if (edit) {
         goto('/providers');
     } else {
         goto('/providers/new');
     }
 })
 form.changeState(state).then((res) => form.form = res);
</script>

{#if $providersSpecs == null}
    <div class="mt-5 d-flex justify-content-center align-items-center gap-2">
        <Spinner color="primary" label={$t('common.spinning')} class="mr-3" /> {$t('wait.retrieving-setting')}
    </div>
{:else}
    <Container fluid class="flex-fill d-flex">
    <Row class="flex-fill">
        <Col lg="4" md="5" class="bg-light">
            <div class="text-center mb-3">
                <ImgProvider
                    {ptype}
                    style="max-width: 100%; max-height: 10em"
                />
            </div>
            <h3>
                {$providersSpecs[ptype].name}
            </h3>

            <p class="text-muted text-justify">
                {$providersSpecs[ptype].description}
            </p>

            {#if form.form && form.form.sideText}
                <hr>
                <p class="text-justify">
                    {form.form.sideText}
                </p>
            {/if}
        </Col>

        <Col lg="8" md="7" class="d-flex flex-column pt-2 pb-3">
            {#if form.form == null}
                <div class="d-flex flex-fill justify-content-center align-items-center gap-2">
                    <Spinner color="primary" label={$t('common.spinning')} class="mr-3" /> {$t('wait.retrieving-setting')}
                </div>
            {:else}
                <form on:submit|preventDefault={() => form.nextState()}>
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
                                specs={{label: $t('provider.name-your'), description: $t('domains.give-explicit-name'), placeholder: $providersSpecs[ptype].name + ' account 1', required: true}}
                                bind:value={form.value._comment}
                                on:input={(event) => form.value._comment = event.detail}
                            />
                        {/if}
                    </CustomForm>
                    <SettingsStateButtons
                        class="d-flex justify-content-end"
                        {edit}
                        form={form.form}
                        nextInProgress={form.nextInProgress}
                        previousInProgress={form.previousInProgress}
                        on:previous-state={() => form.previousState()}
                    />
                </form>
            {/if}
        </Col>
    </Row>
    </Container>
{/if}
