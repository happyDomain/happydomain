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

    import { getAuth, deleteAuthByUid } from '$lib/api-admin';
    import { toasts } from '$lib/stores/toasts';

    let authUsersQ = $state(getAuth());

    let searchQuery = $state('');
</script>

<Container class="flex-fill my-5">
    <Row class="mb-4">
        <Col md={8}>
            <h1 class="display-5">
                <Icon name="shield-lock-fill"></Icon>
                Auth User Management
            </h1>
            <p class="d-flex gap-3 align-items-center text-muted">
                <span class="lead">
                    Manage all authentication accounts
                </span>
                {#await authUsersQ then authUsersR}
                    <span>Total: {authUsersR.data?.length ?? 0} auth users</span>
                {/await}
            </p>
        </Col>
        <Col md={4} class="text-end">
            <Button color="primary" href="/auth_users/new">
                <Icon name="plus-circle"></Icon>
                Create Auth User
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
                    placeholder="Search auth users..."
                    bind:value={searchQuery}
                />
            </InputGroup>
        </Col>
    </Row>

    {#await authUsersQ}
        Please wait...
    {:then authUsersR}
        {@const authUsers = authUsersR.data}
        <div class="table-responsive">
            <Table hover bordered>
                <thead>
                    <tr>
                        <th>ID</th>
                        <th>Email</th>
                        <th>Created</th>
                        <th>Last login</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    {#each (authUsers ?? []).filter(i => (i.id?.toLowerCase() ?? '').indexOf(searchQuery.toLowerCase()) > -1 || (i.email?.toLowerCase() ?? '').indexOf(searchQuery.toLowerCase()) > -1) as authUser}
                        <tr>
                            <td>{authUser.id}</td>
                            <td>{authUser.email}</td>
                            <td>{authUser.createdAt?.replace(/\.[0-9]+/, "") ?? ''}</td>
                            <td>{authUser.lastLoggedIn?.replace(/\.[0-9]+/, "")}</td>
                            <td class="d-flex flex-nowrap gap-1">
                                <Button color="primary" outline size="sm" href="/auth_users/{authUser.id}">
                                    <Icon name="pencil"></Icon>
                                </Button>
                                <Button color="primary" outline size="sm" onclick={async () => {
                                    if (confirm(`Are you sure you want to delete auth user "${authUser.email}"?`)) {
                                        try {
                                            await deleteAuthByUid({ path: { uid: authUser.id ?? '' } });
                                            // Refresh the auth users list
                                            authUsersQ = getAuth();
                                            toasts.addToast({
                                                message: `Auth user "${authUser.email}" has been deleted successfully`,
                                                type: 'success',
                                                timeout: 5000,
                                            });
                                        } catch (error) {
                                            toasts.addErrorToast({
                                                message: 'Failed to delete auth user: ' + error,
                                                timeout: 10000,
                                            });
                                        }
                                    }
                                }}>
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
