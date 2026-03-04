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
    import type { ClassValue } from "svelte/elements";
    import { Spinner } from "@sveltestrap/sveltestrap";

    import PageTitle from "$lib/components/PageTitle.svelte";
    import ImgProvider from "$lib/components/providers/ImgProvider.svelte";
    import PForm from "$lib/components/forms/Provider.svelte";
    import SettingsStateButtons from "$lib/components/providers/SettingsStateButtons.svelte";
    import { ProviderForm } from "$lib/model/provider_form.svelte";
    import type { ProviderSettingsState } from "$lib/model/provider_settings";
    import { navigate } from "$lib/stores/config";
    import { providersSpecs, refreshProviders, refreshProvidersSpecs } from "$lib/stores/providers";
    import { t } from "$lib/translations";

    // Load required data
    if ($providersSpecs == null) refreshProvidersSpecs();

    interface Props {
        class?: ClassValue;
        edit?: boolean;
        ptype: string;
        state: number;
        providerId?: string | null;
        value?: ProviderSettingsState | null;
    }

    let {
        class: className = "",
        edit = false,
        ptype,
        state: formstate,
        providerId = null,
        value = $bindable(null)
    }: Props = $props();

    //
    function createProviderForm(ptype: string, providerId: string | null, value: ProviderSettingsState | null, edit: boolean): ProviderForm {
        const pf = new ProviderForm(
            ptype,
            () => refreshProviders().then(() => navigate("/?newProvider")),
            providerId,
            value,
            () => {
                if (edit) {
                    navigate("/providers");
                } else {
                    navigate("/providers?newProvider");
                }
            },
        );
        pf.changeState(formstate);
        return pf;
    }
    let form: ProviderForm = $derived(createProviderForm(ptype, providerId, value, edit));
</script>

<PageTitle title={$t(value ? "wait.updating" : "provider.new-form")} subtitle={value ? value?._comment : ($providersSpecs ? $providersSpecs[ptype].description : "")}>
    <ImgProvider {ptype} style="max-height: 5rem; width: auto;" />
</PageTitle>
{#if $providersSpecs == null}
    <div class="mt-5 d-flex justify-content-center align-items-center gap-2 {className}">
        <Spinner color="primary" />
        {$t("wait.retrieving-setting")}
    </div>
{:else}
    {#if form.form == null}
        <div class="d-flex flex-fill justify-content-center align-items-center gap-2 {className}">
            <Spinner color="primary" />
            {$t("wait.retrieving-setting")}
        </div>
    {:else}
        <div class={className}>
            <PForm bind:form={form} {ptype} />
            <SettingsStateButtons
                class="d-flex justify-content-end mt-3"
                {edit}
                form={form.form}
                nextInProgress={form.nextInProgress}
                previousInProgress={form.previousInProgress}
                submitForm="providerform"
                on:previous-state={() => form.previousState().then(() => (form = form))}
            />
        </div>
    {/if}
{/if}
