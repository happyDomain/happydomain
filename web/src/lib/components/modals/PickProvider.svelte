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
    } from "@sveltestrap/sveltestrap";

    import ProviderPicker from "$lib/components/forms/ProviderPicker.svelte";
    import SettingsStateButtons from "$lib/components/providers/SettingsStateButtons.svelte";
    import type { Provider } from "$lib/model/provider";
    import type { ProviderForm } from "$lib/model/provider_form.svelte.ts";
    import { providers } from "$lib/stores/providers";
    import { t } from "$lib/translations";

    interface Props {
        ondone?: (provider: Provider) => void;
    }

    let { ondone }: Props = $props();

    let isOpen = $state(false);
    let addingProvider = $state(false);
    let providerType = $state("");
    let form: ProviderForm = $state({} as ProviderForm);

    function Open(): void {
        isOpen = true;
        addingProvider = !$providers || $providers.length === 0;
        providerType = "";
    }

    controls.Open = Open;

    function toggle(): void {
        isOpen = !isOpen;
    }

    function providerSelected(provider: Provider): void {
        isOpen = false;
        ondone?.(provider);
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
        <p>
            {$t("provider.select-provider")}
        </p>
        <ProviderPicker
            bind:addingProvider
            bind:form
            bind:providerType
            formId="pickproviderform"
            ondone={providerSelected}
        />
    </ModalBody>
    <ModalFooter>
        {#if addingProvider && providerType && form}
            <SettingsStateButtons
                canDoNext={form.state >= 0}
                class="d-flex justify-content-end"
                submitForm="pickproviderform"
                form={form.form}
                nextInProgress={form.nextInProgress}
                previousInProgress={form.previousInProgress}
                on:previous-state={previous}
            />
        {:else}
            <Button color="outline-secondary" onclick={toggle}>
                {$t("common.cancel")}
            </Button>
            {#if addingProvider && $providers && $providers.length > 0}
                <Button color="outline-secondary" onclick={previous}>
                    <Icon name="chevron-left" />
                    {$t("common.previous")}
                </Button>
            {/if}
        {/if}
    </ModalFooter>
</Modal>
