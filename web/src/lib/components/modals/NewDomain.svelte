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
        ListGroup,
        ListGroupItem,
        Modal,
        ModalBody,
        ModalFooter,
        ModalHeader,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import DomainImport from "$lib/components/forms/DomainImport.svelte";
    import ProviderConnect from "$lib/components/forms/ProviderConnect.svelte";
    import SettingsStateButtons from "$lib/components/providers/SettingsStateButtons.svelte";
    import ImgProvider from "$lib/components/providers/ImgProvider.svelte";
    import type { Provider } from "$lib/model/provider";
    import type { ProviderForm } from "$lib/model/provider_form.svelte.ts";
    import {
        providers,
        providersSpecs,
        refreshProviders,
        refreshProvidersSpecs,
    } from "$lib/stores/providers";
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

    if (!$providersSpecs) refreshProvidersSpecs();

    function Open(): void {
        isOpen = true;
        step = 0;
        addingProvider = !$providers || $providers.length === 0;
        providerType = "";
        myProvider = {} as Provider;
    }

    controls.Open = Open;

    function toggle(): void {
        isOpen = !isOpen;
    }

    function selectExistingProvider(provider: Provider): void {
        myProvider = provider;
        step = 1;
    }

    function providerAdded(provider: Provider): void {
        refreshProviders();
        myProvider = provider;
        addingProvider = false;
        providerType = "";
        step = 1;
    }

    function previous(): void {
        if (providerType) {
            form.previousState().then(() => {
                if (form.state < 0) {
                    providerType = "";
                } else {
                    form = form;
                }
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
        <p class="text-muted mb-0" style="font-size: 0.85em">
            {$t("domains.add-subtitle")}
        </p>
    </ModalHeader>

    <ModalBody>
        {#if step === 0}
            {#if addingProvider}
                <ProviderConnect
                    bind:form
                    bind:providerType
                    formId="newdomainproviderform"
                    ondone={providerAdded}
                />
            {:else}
                <p>{$t("domains.add-subtitle")}</p>
                <ListGroup>
                    {#each $providers ?? [] as provider (provider._id)}
                        <ListGroupItem
                            tag="button"
                            class="d-flex align-items-center gap-3"
                            onclick={() => selectExistingProvider(provider)}
                        >
                            <ImgProvider
                                id_provider={provider._id}
                                style="max-width: 2em; max-height: 2em; object-fit: contain; flex-shrink: 0;"
                            />
                            <div class="text-start">
                                <div class="fw-semibold">
                                    {#if provider._comment}
                                        {provider._comment}
                                    {:else}
                                        <em>{$t("provider.no-name")}</em>
                                    {/if}
                                </div>
                                <small class="text-muted">
                                    {#if $providersSpecs && $providersSpecs[provider._srctype]}
                                        {$providersSpecs[provider._srctype].name}
                                    {:else}
                                        {provider._srctype}
                                    {/if}
                                </small>
                            </div>
                            <Icon name="chevron-right" class="ms-auto text-muted" />
                        </ListGroupItem>
                    {/each}
                    <ListGroupItem
                        tag="button"
                        class="d-flex align-items-center gap-3 text-primary"
                        onclick={() => (addingProvider = true)}
                    >
                        <Icon name="plus-circle" style="font-size: 1.5em; flex-shrink: 0;" />
                        <div class="text-start fw-semibold">
                            {$t("common.add-new-thing", { thing: $t("provider.kind") })}
                        </div>
                    </ListGroupItem>
                </ListGroup>
            {/if}
        {:else if myProvider && myProvider._id}
            <DomainImport provider={myProvider} />
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
            <Button color="outline-secondary" onclick={() => (step = 0)}>
                <Icon name="chevron-left" />
                {$t("common.previous")}
            </Button>
            <Button color="primary" onclick={toggle}>
                {$t("common.got-it")}
            </Button>
        {/if}
    </ModalFooter>
</Modal>
