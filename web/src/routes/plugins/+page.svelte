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
        Card,
        Col,
        Container,
        Icon,
        Input,
        InputGroup,
        InputGroupText,
        Table,
        Row,
        Badge,
    } from "@sveltestrap/sveltestrap";

    import { t } from '$lib/translations';
    import { listPlugins } from '$lib/api/plugins';

    let pluginsPromise = $state(listPlugins());

    let searchQuery = $state('');
</script>

<svelte:head>
    <title>{$t('plugins.tests.title')} - happyDomain</title>
</svelte:head>

<Container class="flex-fill my-5">
    <Row class="mb-4">
        <Col md={8}>
            <h1 class="display-5">
                <Icon name="check-circle-fill"></Icon>
                {$t('plugins.tests.title')}
            </h1>
            <p class="d-flex gap-3 align-items-center text-muted">
                <span class="lead">
                    {$t('plugins.tests.description')}
                </span>
                {#await pluginsPromise then plugins}
                    <span>{$t('plugins.tests.available-count', { count: Object.keys(plugins ?? {}).length })}</span>
                {/await}
            </p>
        </Col>
    </Row>

    <Row class="mb-4">
        <Col md={8} lg={6}>
            <InputGroup>
                <InputGroupText>
                    <Icon name="search"></Icon>
                </InputGroupText>
                <Input
                    type="text"
                    placeholder={$t('plugins.tests.search-placeholder')}
                    bind:value={searchQuery}
                />
            </InputGroup>
        </Col>
    </Row>

    {#await pluginsPromise}
        <Card body>
            <p class="text-center mb-0">
                <span class="spinner-border spinner-border-sm me-2"></span>
                {$t('plugins.tests.loading')}
            </p>
        </Card>
    {:then plugins}
        <div class="table-responsive">
            <Table hover bordered>
                <thead>
                    <tr>
                        <th>{$t('plugins.tests.table.name')}</th>
                        <th>{$t('plugins.tests.table.version')}</th>
                        <th>{$t('plugins.tests.table.availability')}</th>
                        <th>{$t('plugins.tests.table.actions')}</th>
                    </tr>
                </thead>
                <tbody>
                    {#if !plugins || Object.keys(plugins).length == 0}
                        <tr>
                            <td colspan="4" class="text-center text-muted py-4">
                                {$t('plugins.tests.no-tests')}
                            </td>
                        </tr>
                    {:else}
                        {#each Object.entries(plugins ?? {}).filter(([name, _info]) => name.toLowerCase().indexOf(searchQuery.toLowerCase()) > -1) as [pluginName, pluginInfo]}
                            <tr>
                                <td><strong>{pluginInfo.name || pluginName}</strong></td>
                                <td>{pluginInfo.version}</td>
                                <td>
                                    {#if pluginInfo.availableOn}
                                        {#if pluginInfo.availableOn.applyToDomain}
                                            <Badge color="success">{$t('plugins.tests.availability.domain')}</Badge>
                                        {/if}
                                        {#if pluginInfo.availableOn.limitToProviders && pluginInfo.availableOn.limitToProviders.length > 0}
                                            <Badge color="primary" title={pluginInfo.availableOn.limitToProviders.join(', ')}>
                                                {$t('plugins.tests.availability.provider-specific')}
                                            </Badge>
                                        {/if}
                                        {#if pluginInfo.availableOn.limitToServices && pluginInfo.availableOn.limitToServices.length > 0}
                                            <Badge color="info" title={pluginInfo.availableOn.limitToServices.join(', ')}>
                                                {$t('plugins.tests.availability.service-specific')}
                                            </Badge>
                                        {/if}
                                    {:else}
                                        <Badge color="secondary">{$t('plugins.tests.availability.general')}</Badge>
                                    {/if}
                                </td>
                                <td>
                                    <a href="/plugins/{pluginName}" class="btn btn-sm btn-primary">
                                        <Icon name="gear-fill"></Icon>
                                        {$t('plugins.tests.actions.configure')}
                                    </a>
                                </td>
                            </tr>
                        {/each}
                    {/if}
                </tbody>
            </Table>
        </div>
    {:catch error}
        <Card body color="danger">
            <p class="mb-0">
                <Icon name="exclamation-triangle-fill"></Icon>
                {$t('plugins.tests.error-loading', { error: error.message })}
            </p>
        </Card>
    {/await}
</Container>
