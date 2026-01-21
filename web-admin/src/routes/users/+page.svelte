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

    import { getUsers } from '$lib/api-admin';

    let usersQ = $state(getUsers());

    let searchQuery = $state('');
</script>

<Container class="flex-fill my-5">
    <Row class="mb-4">
        <Col md={8}>
            <h1 class="display-5">
                <Icon name="people-fill"></Icon>
                User Management
            </h1>
            <p class="d-flex gap-3 align-items-center text-muted">
                <span class="lead">
                    Manage all user accounts
                </span>
                {#await usersQ then usersR}
                    <span>Total: {usersR.data?.length ?? 0} users</span>
                {/await}
            </p>
        </Col>
        <Col md={4} class="text-end">
            <Button color="primary">
                <Icon name="plus-circle"></Icon>
                Create User
            </Button>
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
                    placeholder="Search users..."
                    bind:value={searchQuery}
                />
            </InputGroup>
        </Col>
    </Row>

    {#await usersQ}
        Please wait...
    {:then usersR}
        {@const users = usersR.data || []}
        {@const filteredUsers = users.filter(i => {
            const query = searchQuery.toLowerCase();
            return (i.id && i.id.toLowerCase().indexOf(query) > -1) || (i.email && i.email.toLowerCase().indexOf(query) > -1);
        })}
        <div class="table-responsive">
            <Table hover bordered>
                <thead>
                    <tr>
                        <th>ID</th>
                        <th>Email</th>
                        <th>Created</th>
                        <th>Last seen</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    {#each filteredUsers as user}
                        <tr>
                            <td>{user.id}</td>
                            <td>{user.email}</td>
                            <td>{user.created_at?.replace(/\.[0-9]+/, "") || ''}</td>
                            <td>{user.last_seen?.replace(/\.[0-9]+/, "") || ''}</td>
                            <td class="d-flex flex-nowrap gap-1">
                                <Button color="primary" outline size="sm">
                                    <Icon name="pencil"></Icon>
                                </Button>
                                <Button color="primary" outline size="sm">
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
