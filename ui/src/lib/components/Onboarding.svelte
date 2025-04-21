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
        Badge,
        Button,
        Card,
        CardBody,
        CardGroup,
        Col,
        Icon,
        Modal,
        ModalBody,
        ModalFooter,
        ModalHeader,
        Row,
        Spinner,
    } from '@sveltestrap/sveltestrap';

    import CardImportableDomains from '$lib/components/providers/CardImportableDomains.svelte';
    import Logo from '$lib/components/Logo.svelte';
    import NewDomainInput from '$lib/components/NewDomainInput.svelte';
    import PForm from '$lib/components/providers/Form.svelte';
    import ProviderList from '$lib/components/providers/List.svelte';
    import ProviderSelector from '$lib/components/providers/Selector.svelte';
    import SettingsStateButtons from '$lib/components/providers/SettingsStateButtons.svelte';
    import ZoneList from '$lib/components/ZoneList.svelte';
    import type { Provider } from '$lib/model/provider';
    import { ProviderForm } from '$lib/model/provider_form';
    import { domains } from '$lib/stores/domains';
    import { providers, refreshProviders } from '$lib/stores/providers';
    import { t } from '$lib/translations';

    if (!$providers) refreshProviders();

    export let isOpen = false;
    let form: ProviderForm;
    let myDomain: string;
    let myDomainInProgress = false;
    let myProvider: Provider;
    let noDomainsList = false;
    let step = 0;
    let providerType: string;

    function nextPage() {
        if (step == 0 && $providers.length > 0) {
            // A provider already exists, skip adding a new provider
            step += 1;
            myProvider = $providers[0];
        }
        step += 1;
    }

    function providerAdded(event: CustomEvent<Provider>) {
        refreshProviders();
        myProvider = event.detail;
        step += 1;
    }

    function previous() {
        form.previousState().then(() => {
            if (form.state < 0) {
                providerType = "";
                form = null;
            } else {
                form = form;
            }
        });
    }

    function toggle(): void {
        isOpen = !isOpen;
    }
</script>

<Modal
    isOpen={isOpen}
    size="xl"
    scrollable
    {toggle}
>
    <ModalHeader {toggle} class="bg-primary-subtle ps-4 pt-4 align-items-start">
        <h3 class="fw-bolder text-primary mb-1">{$t('common.welcome.start')}<Logo height="30" color="#1cb487" />{$t('common.welcome.end')}</h3>
        <p class="text-muted mb-2" style="font-size: 0.85em">
            {$t('onboarding.intro')}
        </p>
    </ModalHeader>
    <ModalBody class="p-0">
        <Row class="m-0 align-items-start">
            <Col lg={4} xl={3} class="d-none d-lg-block sticky-top p-0">
                <div class="onboarding-steps">
                    <div class="step-item" class:active={step == 0} class:completed={step > 0}>
                        <div class="step-number">
                            {#if step > 0}
                                <Icon name="check" />
                            {:else}
                                1
                            {/if}
                        </div>
                        <div class="step-label">{$t('onboarding.steps.welcome')}</div>
                    </div>
                    <div class="step-item" class:active={step == 1} class:completed={step > 1 && step != 4}>
                        <div class="step-number">
                            {#if step > 1 && step != 4}
                                <Icon name="check" />
                            {:else}
                                2
                            {/if}
                        </div>
                        <div class="step-label">{$t('onboarding.steps.connect')}</div>
                    </div>
                    <div class="step-item" class:active={step == 2} class:completed={step > 2 && step != 4}>
                        <div class="step-number">
                            {#if step > 2 && step != 4}
                                <Icon name="check" />
                            {:else}
                                3
                            {/if}
                        </div>
                        <div class="step-label">{$t('onboarding.steps.import')}</div>
                    </div>
                    <div class="step-item" class:active={step == 3}>
                        <div class="step-number">4</div>
                        <div class="step-label">{$t('onboarding.steps.explore')}</div>
                    </div>
                </div>
            </Col>
            <Col class="p-3">
                {#if step == 0}
                    <h3 class="fw-bolder">{$t('common.welcome.start')}<Logo height="30" />{$t('common.welcome.end')}</h3>
                    <p>
                        {@html $t('onboarding.welcome.purpose', {"happyDomain": `happy<span class="fw-bold">Domain</span>`})}
                    </p>
                    <p>
                        {@html $t('onboarding.welcome.use', {"happyDomain": `happy<span class="fw-bold">Domain</span>`})}
                    </p>
                    <Row cols={{sm: 1, md: 2, xl: 4}}>
                        <Col class="mb-3">
                            <Card body class="h-100">
                                <div class="feature-icon">
                                    <Icon name="globe" />
                                </div>
                                <h4 class="feature-title">{$t('onboarding.welcome.unified.title')}</h4>
                                <p class="feature-description">{$t('onboarding.welcome.unified.description')}</p>
                            </Card>
                        </Col>

                        <Col class="mb-3">
                            <Card body class="h-100">
                                <div class="feature-icon">
                                    <Icon name="balloon-heart" />
                                </div>
                                <h4 class="feature-title">{$t('onboarding.welcome.unified.title')}</h4>
                                <p class="feature-description">{$t('onboarding.welcome.unified.title')}</p>
                            </Card>
                        </Col>

                        <Col class="mb-3">
                            <Card body class="h-100">
                                <div class="feature-icon">
                                    <Icon name="terminal" />
                                </div>
                                <h4 class="feature-title">{$t('onboarding.welcome.unified.title')}</h4>
                                <p class="feature-description">{$t('onboarding.welcome.unified.title')}</p>
                            </Card>
                        </Col>

                        <Col class="mb-3">
                            <Card body class="h-100">
                                <div class="feature-icon">
                                    <Icon name="hand-thumbs-up" />
                                </div>
                                <h4 class="feature-title">{$t('onboarding.welcome.unified.title')}</h4>
                                <p class="feature-description">{$t('onboarding.welcome.unified.title')}</p>
                            </Card>
                        </Col>
                    </Row>
                {:else if step == 1}
                    <h3 class="fw-bolder">{$t('onboarding.connect.title')}</h3>
                    {#if providerType}
                        <PForm
                            bind:form={form}
                            ptype={providerType}
                            on:done={providerAdded}
                        />
                    {:else}
                        <p>
                            {$t('onboarding.connect.intro')}
                        </p>
                        <ProviderSelector
                            on:provider-selected={(event) => providerType = event.detail.ptype}
                        />
                        <p class="mt-3 fw-bold">
                            {$t('onboarding.connect.noprovider')} <a href="https://github.com/happyDomain/happydomain/issues/new" target="_blank" data-umami-event="need-another-provider">{$t('onboarding.connect.noproviderTellUs')}</a>.
                        </p>
                    {/if}
                {:else if step == 2}
                    <h3 class="fw-bolder">{$t('onboarding.import.title')}</h3>
                    <p>
                        {$t('onboarding.import.intro')}
                    </p>
                    <CardImportableDomains
                        provider={myProvider}
                        bind:noDomainsList={noDomainsList}
                    />
                    {#if !myProvider || noDomainsList}
                        <!-- svelte-ignore a11y-autofocus -->
                        <NewDomainInput
                            bind:addingNewDomain={myDomainInProgress}
                            autofocus
                            class="mt-3"
                            id="newDomain"
                            formId="newDomainForm"
                            provider={myProvider}
                            bind:value={myDomain}
                        />
                        {#if $domains}
                            <ZoneList
                                class="mt-3"
                                domains={$domains}
                            >
                                <Badge slot="badges" color="success">
                                    <Icon name="check" />
                                    {$t('onboarding.import.imported')}
                                </Badge>
                            </ZoneList>
                        {/if}
                    {/if}
                {:else if step == 3}
                    {#if $domains.length}
                        <h3 class="text-center display-4">🎉</h3>
                        <h5 class="text-center fw-bolder">{$t('onboarding.explore.done')}</h5>
                        <p class="text-center">
                            {$t('onboarding.explore.bravo', {"count": $domains.length})}
                        </p>
                        <hr class="text-primary">
                    {/if}
                    <h3 class="fw-bolder">{$t('onboarding.explore.title')}</h3>
                    <p>{$t('onboarding.explore.intro')}</p>
                    <Row cols={{sm: 1, md: 2, xl: 4}}>
                        <Col class="mb-3">
                            <Card body class="h-100">
                                <div class="feature-icon">
                                    <Icon name="file-earmark-text" />
                                </div>
                                <h4 class="feature-title">{$t('onboarding.explore.history.title')}</h4>
                                <p class="feature-description">{$t('onboarding.explore.history.description')}</p>
                                {#if $domains.length}
                                    <a href="/domains/{$domains[0].id}/history" class="feature-link">{$t('onboarding.explore.history.link')} <Icon name="arrow-right-short" /></a>
                                {/if}
                            </Card>
                        </Col>

                        <Col class="mb-3">
                            <Card body class="h-100">
                                <div class="feature-icon">
                                    <Icon name="key" />
                                </div>
                                <h4 class="feature-title">{$t('onboarding.explore.api.title')}</h4>
                                <p class="feature-description">{$t('onboarding.explore.api.description')}</p>
                                <a href="/swagger/index.html" class="feature-link">{$t('onboarding.explore.api.link')} <Icon name="arrow-right-short" /></a>
                            </Card>
                        </Col>

                        <Col class="mb-3">
                            <Card body class="h-100">
                                <div class="feature-icon">
                                    <Icon name="clock" />
                                </div>
                                <h4 class="feature-title">{$t('onboarding.explore.monitoring.title')} <small class="text-muted">({$t('onboarding.explore.soon')})</small></h4>
                                <p class="feature-description">{$t('onboarding.explore.monitoring.description')}</p>
                            </Card>
                        </Col>

                        <Col class="mb-3">
                            <Card body class="h-100">
                                <div class="feature-icon">
                                    <Icon name="shield-check" />
                                </div>
                                <h4 class="feature-title">{$t('onboarding.explore.security.title')} <small class="text-muted">({$t('onboarding.explore.soon')})</small></h4>
                                <p class="feature-description">{$t('onboarding.explore.security.description')}</p>
                            </Card>
                        </Col>
                    </Row>
                    <!--li>Edit your zone with ease</li>
                    <li>Share your zone with collaborator</li>
                    <li>Ensure proper configuration/deployment</li-->
                {:else}
                    <h3 class="fw-bolder">
                        {$t('onboarding.no-sale.title')}
                    </h3>
                    <p class="text-justify text-indent mt-4 mb-3">
                        {@html $t('onboarding.no-sale.description', {"happyDomain": `happy<span class="fw-bold">Domain</span>`})}
                    </p>
                    <p class="text-justify text-indent mt-3 mb-4">
                        {$t('onboarding.no-sale.buy-advice')}
                    </p>
                {/if}
            </Col>
        </Row>
    </ModalBody>
    <ModalFooter>
        {#if step == 1 && providerType && form}
            <SettingsStateButtons
                canDoNext={form.state >= 0}
                class="d-flex justify-content-end"
                submitForm="providerform"
                form={form.form}
                nextInProgress={form.nextInProgress}
                previousInProgress={form.previousInProgress}
                on:previous-state={previous}
            />
        {:else}
            {#if step > 0}
                <Button color="outline-secondary" on:click={() => step == 4 ? (step = 1) : (step -= 1)}>
                    <Icon name="chevron-left" class="d-inline d-md-none" />
                    <span class="d-none d-md-inline">{$t('common.previous')}</span>
                </Button>
            {/if}
            {#if step >= 3}
                <Button color="primary" on:click={() => isOpen = false}>
                    {$t('onboarding.explore.understood')}
                </Button>
            {:else if step == 1}
                <Button color="secondary" on:click={() => step = 4}>
                    {$t('onboarding.connect.nodomain')}
                </Button>
            {:else}
                <Button color="primary" on:click={nextPage}>
                    {$t('common.next')}
                </Button>
            {/if}
        {/if}
    </ModalFooter>
</Modal>

<style>
    .onboarding-steps {
        background-color: var(--bs-gray-100);
        border-right: 1px solid var(--bs-gray-200);
        padding: 1.5rem 0;
    }

    .step-item {
        padding: 0.75rem 1.5rem;
        display: flex;
        align-items: center;
        gap: 0.75rem;
        color: var(--bs-gray-600);
        font-size: 0.875rem;
        position: relative;
    }

    .step-item.active {
        color: var(--bs-primary);
        background-color: var(--bs-primary-bg-subtle);
    }

    .step-item.completed {
        color: var(--bs-gray-800);
    }

    .step-number {
        width: 28px;
        height: 28px;
        border-radius: 50%;
        background-color: var(--bs-gray-200);
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 0.75rem;
        font-weight: 600;
        flex-shrink: 0;
    }

    .step-item.active .step-number {
        background-color: var(--bs-primary);
        color: var(--bs-white);
    }

    .step-item.completed .step-number {
        background-color: var(--bs-primary);
        color: var(--bs-white);
    }

    .step-label {
        font-weight: 500;
    }

    .step-content {
        flex: 1;
        padding: 2rem;
        overflow-y: auto;
    }

    .feature-icon {
        width: 48px;
        height: 48px;
        background-color: var(--bs-primary-bg-subtle);
        border-radius: var(--bs-border-radius);
        display: flex;
        align-items: center;
        justify-content: center;
        margin-bottom: 1rem;
        color: var(--primary);
    }

    .feature-title {
        font-size: 1rem;
        font-weight: 600;
        margin-bottom: 0.5rem;
    }


    .feature-description {
        font-size: 0.875rem;
        color: var(--bs-gray-600);
        line-height: 1.5;
        margin-bottom: 1rem;
        flex-grow: 1;
    }

    .feature-link {
        font-size: 0.75rem;
        font-weight: 500;
        color: var(--bs-primary);
        text-decoration: none;
        display: flex;
        align-items: center;
        margin-top: auto;
    }

    .feature-link:hover {
        text-decoration: underline;
    }
</style>
