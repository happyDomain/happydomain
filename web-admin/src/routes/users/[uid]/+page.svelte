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
    import { page } from '$app/stores';
    import {
        Alert,
        Button,
        Col,
        Container,
        Icon,
        Row,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { getUsersByUid, type HappydnsDomain } from '$lib/api-admin';
    import { client } from '$lib/api-admin/client.gen';
    import UserInfoCard from './UserInfoCard.svelte';

    const uid = $page.params.uid!;
    let userQ = $state(getUsersByUid({ path: { uid } }));
</script>

<Container class="flex-fill my-5">
    <Row class="mb-4">
        <Col>
            <h1 class="display-5">
                <Icon name="pencil"></Icon>
                Edit User
            </h1>
        </Col>
    </Row>

    {#await userQ}
        <div class="text-center my-5">
            <Spinner color="primary" />
            <p class="mt-3">Loading user...</p>
        </div>
    {:then userR}
        {@const user = userR.data}
        {#if user}
            <Row>
                <Col md={8} lg={6}>
                    <UserInfoCard {user} {uid} />
                </Col>
            </Row>
        {:else}
            <Alert color="warning">
                <h4 class="alert-heading">User not found</h4>
                <p>The requested user could not be found.</p>
                <hr />
                <Button type="button" color="secondary" outline href="/users">
                    <Icon name="arrow-left"></Icon>
                    Back to Users
                </Button>
            </Alert>
        {/if}
    {:catch error}
        <Alert color="danger">
            <h4 class="alert-heading">Error loading user</h4>
            <p>{error}</p>
            <hr />
            <Button type="button" color="secondary" outline href="/users">
                <Icon name="arrow-left"></Icon>
                Back to Users
            </Button>
        </Alert>
    {/await}
</Container>
