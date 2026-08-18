<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2026 happyDomain
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

<script module lang="ts">
    import type { ModalController } from "$lib/model/modal_controller";

    export const controls: ModalController = {
        Open(): void {},
    };
</script>

<script lang="ts">
    import {
        Button,
        Icon,
        Modal,
        ModalBody,
        ModalFooter,
        ModalHeader,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import DomainImport from "$lib/components/forms/DomainImport.svelte";
    import ProviderPicker from "$lib/components/forms/ProviderPicker.svelte";
    import SettingsStateButtons from "$lib/components/providers/SettingsStateButtons.svelte";
    import type { Provider } from "$lib/model/provider";
    import type { ProviderForm } from "$lib/model/provider_form.svelte.ts";
    import { providers } from "$lib/stores/providers";
    import { t } from "$lib/translations";

    interface Props {
        isOpen?: boolean;
    }

    let { isOpen = $bindable(false) }: Props = $props();

    // step 0: pick or add a provider
    // step 1: import / add domains
    let step = $state(0);

    let addingProvider = $state(false);
    let providerType = $state("");
    let form: ProviderForm = $state({} as ProviderForm);
    let myProvider: Provider = $state({} as Provider);

    let noDomainsList = $state(false);
    let addingNewDomain = $state(false);
    let newDomainValue = $state("");

    function Open(): void {
        isOpen = true;
        step = 0;
        addingProvider = !$providers || $providers.length === 0;
        providerType = "";
        myProvider = {} as Provider;
        noDomainsList = false;
        addingNewDomain = false;
        newDomainValue = "";
    }

    controls.Open = Open;

    function toggle(): void {
        isOpen = !isOpen;
    }

    function onProviderSelected(provider: Provider): void {
        myProvider = provider;
        step = 1;
    }

    function backToProviderPicker(): void {
        step = 0;
        noDomainsList = false;
        addingNewDomain = false;
        newDomainValue = "";
    }

    function previous(): void {
        if (providerType) {
            form.previousState().then(() => {
                if (form.state < 0) providerType = "";
                else form = form;
            });
        } else if (addingProvider && $providers && $providers.length > 0) {
            addingProvider = false;
        } else {
            toggle();
        }
    }
</script>

<Modal {isOpen} scrollable size="lg" {toggle}>
    <ModalHeader {toggle} class="bg-primary-subtle ps-4 pt-4 align-items-start">
        {$t("domains.add-title")}
    </ModalHeader>

    <ModalBody>
        {#if step === 0}
            <p>
                {$t("provider.select-provider")}
            </p>
            <ProviderPicker
                bind:addingProvider
                bind:form
                bind:providerType
                formId="newdomainproviderform"
                ondone={onProviderSelected}
            />
        {:else if myProvider && myProvider._id}
            <DomainImport
                provider={myProvider}
                bind:noDomainsList
                bind:addingNewDomain
                bind:value={newDomainValue}
                formId="newdomaininputform"
                noButton
            />
        {:else}
            <div class="d-flex justify-content-center align-items-center gap-2 my-3">
                <Spinner color="primary" />
            </div>
        {/if}
    </ModalBody>

    <ModalFooter>
        {#if step === 0 && addingProvider && providerType && form}
            <SettingsStateButtons
                canDoNext={form.state >= 0}
                class="d-flex justify-content-end"
                submitForm="newdomainproviderform"
                form={form.form}
                nextInProgress={form.nextInProgress}
                previousInProgress={form.previousInProgress}
                on:previous-state={previous}
            />
        {:else if step === 0}
            <Button color="outline-secondary" onclick={toggle}>
                {$t("common.cancel")}
            </Button>
            {#if addingProvider}
                <Button color="outline-secondary" onclick={previous}>
                    <Icon name="chevron-left" />
                    {$t("common.previous")}
                </Button>
            {/if}
        {:else}
            <Button color="outline-secondary" onclick={backToProviderPicker}>
                <Icon name="chevron-left" />
                {$t("common.previous")}
            </Button>
            {#if noDomainsList}
                <Button
                    color="primary"
                    type="submit"
                    form="newdomaininputform"
                    disabled={!newDomainValue.length || addingNewDomain}
                >
                    {#if addingNewDomain}
                        <Spinner size="sm" class="me-1" />
                    {/if}
                    {$t("common.add-new-thing", { thing: $t("domains.kind") })}
                </Button>
            {:else}
                <Button color="primary" onclick={toggle}>
                    {$t("common.got-it")}
                </Button>
            {/if}
        {/if}
    </ModalFooter>
</Modal>
