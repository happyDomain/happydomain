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
    import { run } from 'svelte/legacy';

    import { Modal, ModalBody, ModalFooter, ModalHeader } from "@sveltestrap/sveltestrap";

    import SettingsStateButtons from "$lib/components/providers/SettingsStateButtons.svelte";
    import ProviderForm from "$lib/components/forms/Provider.svelte";
    import ProviderSelector from "$lib/components/forms/ProviderSelector.svelte";
    import { refreshProviders } from "$lib/stores/providers";
    import { t } from "$lib/translations";

    interface Props {
        isOpen?: boolean;
    }

    let { isOpen = $bindable(false) }: Props = $props();
    run(() => {
        if (isOpen) ptype = "";
    });

    let form: ProviderForm = $state({} as ProviderForm);
    let ptype: string = $state("");

    function previous() {
        if (!form || form.state < 0) {
            isOpen = false;
        } else {
            form.previousState().then(() => {
                if (form.state < 0) {
                    ptype = "";
                } else {
                    form = form;
                }
            });
        }
    }

    function selectProvider(event: CustomEvent<{ ptype: string }>) {
        ptype = event.detail.ptype;
    }

    function finished() {
        isOpen = false;
        refreshProviders();
    }

    function toggle(): void {
        isOpen = !isOpen;
    }
</script>

<Modal {isOpen} scrollable size="lg" {toggle}>
    <ModalHeader {toggle} class="bg-primary-subtle ps-4 pt-4 align-items-start">
        {$t("provider.new-form")}
    </ModalHeader>
    <ModalBody>
        {#if !ptype}
            <p>
                {$t("provider.select-provider")}
            </p>
            <ProviderSelector on:provider-selected={selectProvider} />
        {:else}
            <ProviderForm bind:form={form} formId="providermodal" {ptype} on:done={finished} />
        {/if}
    </ModalBody>
    <ModalFooter>
        {#if ptype && form}
            <SettingsStateButtons
                canDoNext={form && form.state >= 0}
                class="d-flex justify-content-end"
                submitForm="providermodal"
                form={form.form}
                nextInProgress={form.nextInProgress}
                previousInProgress={form.previousInProgress}
                on:previous-state={previous}
            />
        {:else}
            <SettingsStateButtons
                canDoNext={false}
                class="d-flex justify-content-end"
                submitForm="providermodal"
                on:previous-state={previous}
            />
        {/if}
    </ModalFooter>
</Modal>
