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

    import { getDomains, deleteDomainsByDomain } from '$lib/api-admin';
    import { toasts } from '$lib/stores/toasts';

    let domainsQ = $state(getDomains());

    let searchQuery = $state('');

    async function handleDeleteDomain(domainId: string, domainName: string) {
        if (confirm(`Are you sure you want to delete domain "${domainName}"?`)) {
            try {
                await deleteDomainsByDomain({ path: { domain: domainId } });
                // Refresh the domains list
                domainsQ = getDomains();
                toasts.addToast({
                    message: `Domain "${domainName}" has been deleted successfully`,
                    type: 'success',
                    timeout: 5000,
                });
            } catch (error) {
                toasts.addErrorToast({
                    message: 'Failed to delete domain: ' + error,
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
                <Icon name="globe"></Icon>
                Domain Management
            </h1>
            <p class="d-flex gap-3 align-items-center text-muted">
                <span class="lead">
                    Manage all domains
                </span>
                {#await domainsQ then domainsR}
                    <span>Total: {domainsR.data?.length ?? 0} domains</span>
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
                    placeholder="Search domains..."
                    bind:value={searchQuery}
                />
            </InputGroup>
        </Col>
    </Row>

    {#await domainsQ}
        Please wait...
    {:then domainsR}
        {@const domains = domainsR.data ?? []}
        <div class="table-responsive">
            <Table hover bordered>
                <thead>
                    <tr>
                        <th>ID</th>
                        <th>Domain Name</th>
                        <th>Group</th>
                        <th>Owner ID</th>
                        <th>Provider ID</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    {#each domains.filter(d =>
                        (d.id && d.id.toLowerCase().indexOf(searchQuery.toLowerCase()) > -1) ||
                        (d.domain && d.domain.toLowerCase().indexOf(searchQuery.toLowerCase()) > -1) ||
                        (d.group && d.group.toLowerCase().indexOf(searchQuery.toLowerCase()) > -1)
                    ) as domain}
                        <tr>
                            <td>{domain.id}</td>
                            <td>{domain.domain}</td>
                            <td>{domain.group || '-'}</td>
                            <td>{domain.id_owner || '-'}</td>
                            <td>{domain.id_provider || '-'}</td>
                            <td class="d-flex flex-nowrap gap-1">
                                <Button color="primary" outline size="sm" href="/users/{domain.id_owner}/domains/{domain.id}">
                                    <Icon name="pencil"></Icon>
                                </Button>
                                <Button color="primary" outline size="sm" onclick={() => handleDeleteDomain(domain.id || '', domain.domain || '')}>
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
