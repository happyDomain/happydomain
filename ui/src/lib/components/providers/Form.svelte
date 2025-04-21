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
    import { createEventDispatcher, onMount } from 'svelte';

    import CustomForm from '$lib/components/CustomForm.svelte';
    import ResourceInput from '$lib/components/resources/basic.svelte';
    import { ProviderForm } from '$lib/model/provider_form';
    import { providersSpecs } from '$lib/stores/providers';
    import { t } from '$lib/translations';

    const dispatch = createEventDispatcher();

    export let formId = "providerform"
    export let form: ProviderForm;
    export let ptype: string;

    function newForm(ptype) {
        form = new ProviderForm(ptype, (provider) => dispatch("done", provider));
        if (ptype) {
            form.changeState(0).then((res) => {
                form.form = res;
            });
        }
    }

    $: if (!form || form.ptype != ptype) newForm(ptype);
</script>

<form
    id={formId}
    on:submit|preventDefault={() => form.nextState().then(() => form = form)}
>
    {#if form && form.form}
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
