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
        Badge,
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

    // This demonstrates using shared $lib components
    // import { api } from '$lib/api';

    let users = [
        { id: 1, email: 'admin@example.com', created: '2024-01-15', status: 'active' },
        { id: 2, email: 'user@example.com', created: '2024-02-20', status: 'active' },
    ];

    let searchQuery = $state('');

    // TODO: Load users from admin API
    // async function loadUsers() {
    //   const response = await fetch('/admin-api/users');
    //   users = await response.json();
    // }
</script>

<Container class="flex-fill my-5">
    <Row class="mb-4">
        <Col md={8}>
            <h1 class="display-5">
                <Icon name="people-fill"></Icon>
                User Management
            </h1>
            <p class="text-muted">Manage all user accounts</p>
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

    <div class="table-responsive">
        <Table hover bordered>
            <thead>
                <tr>
                    <th>ID</th>
                    <th>Email</th>
                    <th>Created</th>
                    <th>Status</th>
                    <th>Actions</th>
                </tr>
            </thead>
            <tbody>
                {#each users as user}
                    <tr>
                        <td>{user.id}</td>
                        <td>{user.email}</td>
                        <td>{user.created}</td>
                        <td>
                            <Badge color="success">{user.status}</Badge>
                        </td>
                        <td>
                            <Button color="primary" outline size="sm" class="me-1">
                                <Icon name="pencil"></Icon>
                            </Button>
                            <Button color="primary" outline size="sm" class="me-1">
                                <Icon name="trash"></Icon>
                            </Button>
                        </td>
                    </tr>
                {/each}
            </tbody>
        </Table>
    </div>
</Container>
