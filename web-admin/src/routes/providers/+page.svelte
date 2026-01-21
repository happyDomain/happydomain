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
        Button,
        Card,
        Col,
        Container,
        Icon,
        Input,
        InputGroup,
        InputGroupText,
        Table,
        Row,
    } from "@sveltestrap/sveltestrap";

    import { getProviders, deleteProvidersByPid } from '$lib/api-admin';
    import { toasts } from '$lib/stores/toasts';

    let providersQ = $state(getProviders());

    let searchQuery = $state('');

    async function handleDeleteProvider(providerId: string, providerName: string) {
        if (confirm(`Are you sure you want to delete provider "${providerName}"?`)) {
            try {
                await deleteProvidersByPid({ path: { pid: providerId } });
                // Refresh the providers list
                providersQ = getProviders();
                toasts.addToast({
                    message: `Provider "${providerName}" has been deleted successfully`,
                    type: 'success',
                    timeout: 5000,
                });
            } catch (error) {
                toasts.addErrorToast({
                    message: 'Failed to delete provider: ' + error,
                    timeout: 10000,
                });
            }
        }
    }
</script>

<Container class="flex-fill my-5">
    <Row class="mb-4">
        <Col md={8}>
            <h1 class="display-5">
                <Icon name="cloud"></Icon>
                Provider Management
            </h1>
            <p class="d-flex gap-3 align-items-center text-muted">
                <span class="lead">
                    Manage all DNS providers
                </span>
                {#await providersQ then providersR}
                    <span>Total: {providersR.data?.length || 0} providers</span>
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
                    placeholder="Search providers..."
                    bind:value={searchQuery}
                />
            </InputGroup>
        </Col>
    </Row>

    {#await providersQ}
        Please wait...
    {:then providersR}
        {@const providers = providersR.data || []}
        <div class="table-responsive">
            <Table hover bordered>
                <thead>
                    <tr>
                        <th>ID</th>
                        <th>Name</th>
                        <th>Type</th>
                        <th>Owner ID</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    {#each providers.filter(p =>
                        (p._id && p._id.toLowerCase().indexOf(searchQuery.toLowerCase()) > -1) ||
                        (p._comment && p._comment.toLowerCase().indexOf(searchQuery.toLowerCase()) > -1) ||
                        (p._srctype && p._srctype.toLowerCase().indexOf(searchQuery.toLowerCase()) > -1) ||
                        (p._ownerid && p._ownerid.toLowerCase().indexOf(searchQuery.toLowerCase()) > -1)
                    ) as provider}
                        <tr>
                            <td>{provider._id}</td>
                            <td>{provider._comment || '-'}</td>
                            <td>{provider._srctype || '-'}</td>
                            <td>{provider._ownerid || '-'}</td>
                            <td class="d-flex flex-nowrap gap-1">
                                <Button color="primary" outline size="sm" href="/users/{provider._ownerid}/providers/{provider._id}">
                                    <Icon name="pencil"></Icon>
                                </Button>
                                <Button color="primary" outline size="sm" onclick={() => handleDeleteProvider(provider._id || '', provider._comment || provider._id || '')}>
                                    <Icon name="trash"></Icon>
                                </Button>
                            </td>
                        </tr>
                    {/each}
                </tbody>
            </Table>
        </div>
    {/await}
</Container>
