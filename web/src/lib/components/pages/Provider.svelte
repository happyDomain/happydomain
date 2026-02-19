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
    import { navigate } from "$lib/stores/config";

    import { Button, Col, Container, Row, Spinner } from "@sveltestrap/sveltestrap";

    import ImgProvider from "$lib/components/providers/ImgProvider.svelte";
    import PForm from "$lib/components/forms/Provider.svelte";
    import SettingsStateButtons from "$lib/components/providers/SettingsStateButtons.svelte";
    import { ProviderForm } from "$lib/model/provider_form.svelte";
    import type { ProviderSettingsState } from "$lib/model/provider_settings";
    import { providersSpecs, refreshProviders, refreshProvidersSpecs } from "$lib/stores/providers";
    import { t } from "$lib/translations";

    // Load required data
    if ($providersSpecs == null) refreshProvidersSpecs();

    interface Props {
        // Inputs
        edit?: boolean;
        ptype: string;
        state: number;
        providerId?: string | null;
        value?: ProviderSettingsState | null;
    }

    let {
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
                    navigate("/providers/new");
                }
            },
        );
        pf.changeState(formstate);
        return pf;
    }
    let form: ProviderForm = $derived(createProviderForm(ptype, providerId, value, edit));
</script>

{#if $providersSpecs == null}
    <div class="mt-5 d-flex justify-content-center align-items-center gap-2">
        <Spinner color="primary" />
        {$t("wait.retrieving-setting")}
    </div>
{:else}
    <Container fluid class="flex-fill d-flex">
        <Row class="flex-fill">
            <Col lg="4" md="5" style="background-color: #edf5f2;">
                <div class="text-center mb-3">
                    <ImgProvider {ptype} style="max-width: 100%; max-height: 10em" />
                </div>
                <h3>
                    {$providersSpecs[ptype].name}
                </h3>

                <p class="text-muted text-justify">
                    {$providersSpecs[ptype].description}
                </p>

                {#if form.form && form.form.sideText}
                    <hr />
                    <p class="text-justify">
                        {form.form.sideText}
                    </p>
                {/if}
            </Col>

            <Col lg="8" md="7" class="d-flex flex-column pt-2 pb-3">
                {#if form.form == null}
                    <div class="d-flex flex-fill justify-content-center align-items-center gap-2">
                        <Spinner color="primary" />
                        {$t("wait.retrieving-setting")}
                    </div>
                {:else}
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
                {/if}
            </Col>
        </Row>
    </Container>
{/if}
