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
    import { page } from "$app/state";
    import { Alert, Button, Col, Container, Icon, Row, Spinner } from "@sveltestrap/sveltestrap";

    import {
        getUsersByUidProvidersByPid,
        getUsersByUidProvidersByPidDomains,
    } from "$lib/api-admin";
    import ProviderInfoCard from "./ProviderInfoCard.svelte";
    import UserDomainsCard from "../../domains/UserDomainsCard.svelte";

    let uid = $derived(page.params.uid!);
    let provider = $derived(page.params.provider!);
    let providerQ = $derived(getUsersByUidProvidersByPid({ path: { uid, pid: provider } }));
    let domainsQ = $derived(getUsersByUidProvidersByPidDomains({ path: { uid, pid: provider } }));
</script>

<Container class="flex-fill my-5">
    <Row class="mb-4">
        <Col>
            <h1 class="display-5">
                <Icon name="cloud"></Icon>
                Edit Provider
            </h1>
        </Col>
    </Row>

    {#await providerQ}
        <div class="text-center my-5">
            <Spinner color="primary" />
            <p class="mt-3">Loading provider...</p>
        </div>
    {:then providerR}
        {@const providerData = providerR.data}
        {#if providerData}
            <Row>
                <Col md={8} lg={6}>
                    <ProviderInfoCard provider={providerData} {uid} />
                </Col>

                <Col md={8} lg={6}>
                    <UserDomainsCard {domainsQ} userId={providerData._ownerid!} />
                </Col>
            </Row>
        {:else}
            <Alert color="warning">
                <h4 class="alert-heading">Provider not found</h4>
                <p>The requested provider could not be loaded.</p>
                <hr />
                <Button type="button" color="secondary" outline href="/users/{uid}">
                    Back to user
                </Button>
            </Alert>
        {/if}
    {:catch error}
        <Alert color="danger">
            <h4 class="alert-heading">Error loading provider</h4>
            <p>{error}</p>
            <hr />
            <Button type="button" color="secondary" outline href="/users/{uid}">
                Back to user
            </Button>
        </Alert>
    {/await}
</Container>
