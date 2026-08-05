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
    import type { Provider } from "$lib/model/provider";

    export interface NewDomainRequest {
        // ondone, when set, is handed the provider the user picked, and the
        // dialog closes there: for callers already knowing the domain to add.
        // Without it, the dialog goes on with importing domains from the
        // provider.
        ondone?: (provider: Provider) => void;
    }

    // The dialog is mounted once, by each page opening it: the home page and
    // the domains list.
    export const controls = {
        Open(_request?: NewDomainRequest): void {},
    };
</script>

<script lang="ts">
    import { goto } from "$app/navigation";
    import { resolve } from "$app/paths";
    import {
        Button,
        Icon,
        ModalBody,
        ModalFooter,
        ModalHeader,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import Modal from "$lib/components/Modal.svelte";

    import DomainImport from "$lib/components/forms/DomainImport.svelte";
    import ProviderPicker from "$lib/components/forms/ProviderPicker.svelte";
    import SettingsStateButtons from "$lib/components/providers/SettingsStateButtons.svelte";
    import { updateProvider } from "$lib/api/provider";
    import type { ProviderForm } from "$lib/model/provider_form.svelte.ts";
    import { providers, refreshProviders } from "$lib/stores/providers";
    import { toasts } from "$lib/stores/toasts";
    import { t } from "$lib/translations";

    // The read-only provider recomposing the zone of domains hosted where
    // happyDomain has no access.
    const discoveryProviderType = "PublicDNSDiscovery";

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
    let pickingDiscovery = $state(false);

    let isOpen = $state(false);
    let request: NewDomainRequest = {};

    function Open(newRequest: NewDomainRequest = {}): void {
        isOpen = true;
        request = newRequest;
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
        if (request.ondone) {
            isOpen = false;
            request.ondone(provider);
            return;
        }
        myProvider = provider;
        step = 1;
    }

    // monitorElsewhere goes on with the discovery provider of the user,
    // created on first use: it has nothing to configure.
    async function monitorElsewhere(): Promise<void> {
        pickingDiscovery = true;
        try {
            // Load the providers first: a user already having one would get a second.
            if (!$providers) await refreshProviders();
            let provider = $providers?.find((p) => p._srctype === discoveryProviderType);
            if (!provider) {
                provider = await updateProvider({
                    _srctype: discoveryProviderType,
                    _comment: $t("domains.monitor-elsewhere.provider-name"),
                    Provider: {},
                } as Provider);
                await refreshProviders();
            }
            onProviderSelected(provider);
        } catch (err) {
            toasts.addErrorToast({
                message: err instanceof Error ? err.message : String(err),
            });
        } finally {
            pickingDiscovery = false;
        }
    }

    function backToProviderPicker(): void {
        step = 0;
        noDomainsList = false;
        addingNewDomain = false;
        newDomainValue = "";
    }

    // onDomainAdded takes the user to the domain typed in by hand: it is the
    // one they came to add.
    function onDomainAdded(dn: string): void {
        isOpen = false;
        goto(resolve("/domains/[dn]/[[historyid]]", { dn }));
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
            {#if !providerType}
                <hr />
                <p class="text-muted mb-2">
                    {$t("domains.monitor-elsewhere.hint")}
                </p>
                <Button
                    color="outline-secondary"
                    disabled={pickingDiscovery}
                    onclick={monitorElsewhere}
                >
                    {#if pickingDiscovery}
                        <Spinner size="sm" class="me-1" />
                    {:else}
                        <Icon name="binoculars" class="me-1" />
                    {/if}
                    {$t("domains.monitor-elsewhere.button")}
                </Button>
            {/if}
        {:else if myProvider && myProvider._id}
            <DomainImport
                provider={myProvider}
                bind:noDomainsList
                bind:addingNewDomain
                bind:value={newDomainValue}
                formId="newdomaininputform"
                noButton
                onadded={(domain) => onDomainAdded(domain.domain)}
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
