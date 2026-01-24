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

    import { getPluginsTests } from '$lib/api-admin';

    let pluginsQ = $state(getPluginsTests());

    let searchQuery = $state('');
</script>

<Container class="flex-fill my-5">
    <Row class="mb-4">
        <Col md={8}>
            <h1 class="display-5">
                <Icon name="puzzle-fill"></Icon>
                Plugins Management
            </h1>
            <p class="d-flex gap-3 align-items-center text-muted">
                <span class="lead">
                    Manage all test plugins
                </span>
                {#await pluginsQ then pluginsR}
                    <span>Total: {Object.keys(pluginsR.data ?? {}).length} plugins</span>
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
                    placeholder="Search plugins..."
                    bind:value={searchQuery}
                />
            </InputGroup>
        </Col>
    </Row>

    {#await pluginsQ}
        Please wait...
    {:then pluginsR}
        {@const plugins = pluginsR.data}
        <div class="table-responsive">
            <Table hover bordered>
                <thead>
                    <tr>
                        <th>Plugin Name</th>
                        <th>Version</th>
                        <th>Availability</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    {#if !plugins || Object.keys(plugins).length == 0}
                        <tr>
                            <td colspan="4" class="text-center text-muted py-2">
                                No plugins available
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
                                            <Badge color="success">Domain</Badge>
                                        {/if}
                                        {#if pluginInfo.availableOn.limitToProviders && pluginInfo.availableOn.limitToProviders.length > 0}
                                            <Badge color="primary" title={pluginInfo.availableOn.limitToProviders.join(', ')}>
                                                Provider-specific
                                            </Badge>
                                        {/if}
                                        {#if pluginInfo.availableOn.limitToServices && pluginInfo.availableOn.limitToServices.length > 0}
                                            <Badge color="info" title={pluginInfo.availableOn.limitToServices.join(', ')}>
                                                Service-specific
                                            </Badge>
                                        {/if}
                                    {:else}
                                        <Badge color="secondary">General</Badge>
                                    {/if}
                                </td>
                                <td>
                                    <a href="/plugins/{pluginName}" class="btn btn-sm btn-primary">
                                        <Icon name="gear-fill"></Icon>
                                        Manage
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
                Error loading plugins: {error.message}
            </p>
        </Card>
    {/await}
</Container>
