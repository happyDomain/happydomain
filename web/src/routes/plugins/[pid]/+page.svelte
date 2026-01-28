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

<script lang="ts">
    import {
        Alert,
        Badge,
        Button,
        Card,
        CardBody,
        CardHeader,
        Col,
        Container,
        Form,
        FormGroup,
        Icon,
        Row,
    } from "@sveltestrap/sveltestrap";
    import { page } from '$app/stores';

    import { t } from '$lib/translations';
    import { toasts } from '$lib/stores/toasts';
    import {
        getPluginStatus,
        getPluginOptions,
        updatePluginOptions,
    } from '$lib/api/plugins';
    import Resource from '$lib/components/inputs/Resource.svelte';

    let pid = $derived($page.params.pid!);

    let pluginStatusPromise = $derived(getPluginStatus(pid));
    let pluginOptionsPromise = $derived(getPluginOptions(pid));
    let optionValues = $state<Record<string, any>>({});
    let saving = $state(false);

    $effect(() => {
        pluginOptionsPromise.then((options) => {
            optionValues = { ...(options || {}) };
        });
    });

    async function saveOptions() {
        saving = true;
        try {
            await updatePluginOptions(pid, optionValues);
            pluginOptionsPromise = getPluginOptions(pid);
            toasts.addToast({
                message: $t('plugins.tests.messages.options-updated'),
                type: 'success',
                timeout: 5000,
            });
        } catch (error) {
            toasts.addErrorToast({
                message: $t('plugins.tests.messages.update-failed', { error: String(error) }),
                timeout: 10000,
            });
        } finally {
            saving = false;
        }
    }

    async function cleanOrphanedOptions(userOpts: any[]) {
        const validOptIds = new Set(userOpts.map(opt => opt.id));
        const cleanedOptions: Record<string, any> = {};

        for (const [key, value] of Object.entries(optionValues)) {
            if (validOptIds.has(key)) {
                cleanedOptions[key] = value;
            }
        }

        saving = true;
        try {
            await updatePluginOptions(pid, cleanedOptions);
            pluginOptionsPromise = getPluginOptions(pid);
            toasts.addToast({
                message: $t('plugins.tests.messages.options-cleaned'),
                type: 'success',
                timeout: 5000,
            });
        } catch (error) {
            toasts.addErrorToast({
                message: $t('plugins.tests.messages.clean-failed', { error: String(error) }),
                timeout: 10000,
            });
        } finally {
            saving = false;
        }
    }

    function getOrphanedOptions(userOpts: any[]): string[] {
        const validOptIds = new Set(userOpts.map(opt => opt.id));
        return Object.keys(optionValues).filter(key => !validOptIds.has(key));
    }
</script>

<svelte:head>
    <title>{pid} - {$t('plugins.tests.title')} - happyDomain</title>
</svelte:head>

<Container class="flex-fill my-5">
    <Row class="mb-4">
        <Col>
            <Button color="link" href="/plugins" class="mb-2">
                <Icon name="arrow-left"></Icon>
                {$t('plugins.tests.back-to-tests')}
            </Button>
            <h1 class="display-5">
                <Icon name="check-circle-fill"></Icon>
                {pid}
            </h1>
        </Col>
    </Row>

    {#await pluginStatusPromise}
        <Card body>
            <p class="text-center mb-0">
                <span class="spinner-border spinner-border-sm me-2"></span>
                {$t('plugins.tests.loading-info')}
            </p>
        </Card>
    {:then status}
        {#if status}
            <Row class="mb-4">
                <Col md={6}>
                    <Card>
                        <CardHeader>
                            <strong>{$t('plugins.tests.detail.test-information')}</strong>
                        </CardHeader>
                        <CardBody>
                            <dl class="row mb-0">
                                <dt class="col-sm-4">{$t('plugins.tests.detail.name')}</dt>
                                <dd class="col-sm-8">{status.name}</dd>

                                <dt class="col-sm-4">{$t('plugins.tests.detail.version')}</dt>
                                <dd class="col-sm-8">{status.version}</dd>

                                <dt class="col-sm-4">{$t('plugins.tests.detail.availability')}</dt>
                                <dd class="col-sm-8">
                                    {#if status.availableOn}
                                        <div class="d-flex flex-wrap gap-1">
                                            {#if status.availableOn.applyToDomain}
                                                <Badge color="success">{$t('plugins.tests.availability.domain-level')}</Badge>
                                            {/if}
                                            {#if status.availableOn.limitToProviders && status.availableOn.limitToProviders.length > 0}
                                                <Badge color="primary">
                                                    {$t('plugins.tests.availability.providers', { providers: status.availableOn.limitToProviders.join(', ') })}
                                                </Badge>
                                            {/if}
                                            {#if status.availableOn.limitToServices && status.availableOn.limitToServices.length > 0}
                                                <Badge color="info">
                                                    {$t('plugins.tests.availability.services', { services: status.availableOn.limitToServices.join(', ') })}
                                                </Badge>
                                            {/if}
                                            {#if !status.availableOn.applyToDomain &&
                                                 (!status.availableOn.limitToProviders || status.availableOn.limitToProviders.length === 0) &&
                                                 (!status.availableOn.limitToServices || status.availableOn.limitToServices.length === 0)}
                                                <Badge color="secondary">{$t('plugins.tests.availability.general')}</Badge>
                                            {/if}
                                        </div>
                                    {:else}
                                        <Badge color="secondary">{$t('plugins.tests.availability.general')}</Badge>
                                    {/if}
                                </dd>
                            </dl>
                        </CardBody>
                    </Card>
                </Col>

                <Col md={6}>
                    {#await pluginOptionsPromise}
                        <Card>
                            <CardBody>
                                <p class="text-center mb-0">
                                    <span class="spinner-border spinner-border-sm me-2"></span>
                                    {$t('plugins.tests.detail.loading-options')}
                                </p>
                            </CardBody>
                        </Card>
                    {:then options}
                        {@const userOpts = status.options?.userOpts || []}
                        {@const readOnlyOptGroups = [
                            { key: 'adminOpts', label: $t('plugins.tests.option-groups.global-settings'), opts: status.options?.adminOpts || [] },
                            { key: 'domainOpts', label: $t('plugins.tests.option-groups.domain-settings'), opts: status.options?.domainOpts || [] },
                            { key: 'serviceOpts', label: $t('plugins.tests.option-groups.service-settings'), opts: status.options?.serviceOpts || [] },
                            { key: 'runOpts', label: $t('plugins.tests.option-groups.test-parameters'), opts: status.options?.runOpts || [] }
                          ]}
                        {@const hasAnyOpts = userOpts.length > 0 || readOnlyOptGroups.some(g => g.opts.length > 0)}
                        {@const orphanedOpts = getOrphanedOptions(userOpts)}

                        {#if orphanedOpts.length > 0}
                            <Alert color="warning" class="mb-3">
                                <div class="d-flex justify-content-between align-items-center">
                                    <div>
                                        <Icon name="exclamation-triangle-fill"></Icon>
                                        {$t('plugins.tests.detail.orphaned-options', { options: orphanedOpts.join(', ') })}
                                    </div>
                                    <Button
                                        color="danger"
                                        size="sm"
                                        onclick={() => cleanOrphanedOptions(userOpts)}
                                        disabled={saving}
                                    >
                                        <Icon name="trash"></Icon>
                                        {$t('plugins.tests.detail.clean-up')}
                                    </Button>
                                </div>
                            </Alert>
                        {/if}

                        {#if userOpts.length > 0}
                            <Card class="mb-3">
                                <CardHeader>
                                    <strong>{$t('plugins.tests.detail.configuration')}</strong>
                                </CardHeader>
                                <CardBody>
                                    <Form on:submit={(e) => { e.preventDefault(); saveOptions(); }}>
                                        {#each userOpts as optDoc}
                                            {#if optDoc.id}
                                                {@const optName = optDoc.id}
                                                <FormGroup>
                                                    <Resource
                                                        edit={true}
                                                        index={optName}
                                                        specs={optDoc}
                                                        type={optDoc.type || 'string'}
                                                        bind:value={optionValues[optName]}
                                                    />
                                                </FormGroup>
                                            {/if}
                                        {/each}
                                        <div class="d-flex gap-2">
                                            <Button type="submit" color="success" disabled={saving}>
                                                {#if saving}
                                                    <span class="spinner-border spinner-border-sm me-1"></span>
                                                {/if}
                                                <Icon name="check-circle"></Icon>
                                                {$t('plugins.tests.detail.save-changes')}
                                            </Button>
                                        </div>
                                    </Form>
                                </CardBody>
                            </Card>
                        {/if}

                        {#each readOnlyOptGroups as optGroup}
                            {#if optGroup.opts.length > 0}
                                <Card class="mb-3">
                                    <CardHeader>
                                        <strong>{optGroup.label}</strong>
                                        <small class="text-muted ms-2">{$t('plugins.tests.detail.read-only')}</small>
                                    </CardHeader>
                                    <CardBody>
                                        <dl class="row mb-0">
                                            {#each optGroup.opts as optDoc}
                                                <dt class="col-sm-4">{optDoc.label || optDoc.id}:</dt>
                                                <dd class="col-sm-8">
                                                    {#if optDoc.default}
                                                        <span class="text-muted d-block">{optDoc.default}</span>
                                                    {:else if optDoc.placeholder}
                                                        <em class="text-muted d-block">{optDoc.placeholder}</em>
                                                    {/if}
                                                    {#if optDoc.description}
                                                        <small class="text-muted d-block">{optDoc.description}</small>
                                                    {/if}
                                                    <small class="text-muted">{$t('plugins.tests.option-groups.type', { type: optDoc.type || 'string' })}</small>
                                                    {#if optDoc.required}<small class="text-danger ms-2">{$t('plugins.tests.option-groups.required')}</small>{/if}
                                                </dd>
                                            {/each}
                                        </dl>
                                    </CardBody>
                                </Card>
                            {/if}
                        {/each}

                        {#if !hasAnyOpts}
                            <Card>
                                <CardBody>
                                    <Alert color="info" class="mb-0">
                                        <Icon name="info-circle"></Icon>
                                        {$t('plugins.tests.detail.no-configurable-options')}
                                    </Alert>
                                </CardBody>
                            </Card>
                        {/if}
                    {:catch error}
                        <Card>
                            <CardBody>
                                <Alert color="danger" class="mb-0">
                                    <Icon name="exclamation-triangle-fill"></Icon>
                                    {$t('plugins.tests.detail.error-loading-options', { error: error.message })}
                                </Alert>
                            </CardBody>
                        </Card>
                    {/await}
                </Col>
            </Row>
        {:else}
            <Alert color="danger">
                <Icon name="exclamation-triangle-fill"></Icon>
                {$t('plugins.tests.test-info-not-found')}
            </Alert>
        {/if}
    {:catch error}
        <Alert color="danger">
            <Icon name="exclamation-triangle-fill"></Icon>
            {$t('plugins.tests.error-loading-test', { error: error.message })}
        </Alert>
    {/await}
</Container>
