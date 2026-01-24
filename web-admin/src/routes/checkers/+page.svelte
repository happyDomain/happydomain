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

    import { getChecks } from "$lib/api-admin";

    let checkersQ = $state(getChecks());

    let searchQuery = $state("");
</script>

<Container class="flex-fill my-5">
    <Row class="mb-4">
        <Col md={8}>
            <h1 class="display-5">
                <Icon name="puzzle-fill"></Icon>
                Checkers
            </h1>
            <p class="d-flex gap-3 align-items-center text-muted">
                <span class="lead"> Manage all checkers </span>
                {#await checkersQ then checkersR}
                    <span>Total: {Object.keys(checkersR.data ?? {}).length} checkers</span>
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
                <Input type="text" placeholder="Search checker..." bind:value={searchQuery} />
            </InputGroup>
        </Col>
    </Row>

    {#await checkersQ}
        Please wait...
    {:then checkersR}
        {@const checkers = checkersR.data}
        <div class="table-responsive">
            <Table hover bordered>
                <thead>
                    <tr>
                        <th>Plugin Name</th>
                        <th>Availability</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    {#if !checkers || Object.keys(checkers).length == 0}
                        <tr>
                            <td colspan="4" class="text-center text-muted py-2">
                                No checkers available
                            </td>
                        </tr>
                    {:else}
                        {#each Object.entries(checkers ?? {}).filter(([name, _info]) => name
                                    .toLowerCase()
                                    .indexOf(searchQuery.toLowerCase()) > -1) as [checkerName, checkerInfo]}
                            <tr>
                                <td><strong>{checkerInfo.name || checkerName}</strong></td>
                                <td>
                                    {#if checkerInfo.availability}
                                        {#if checkerInfo.availability.applyToDomain}
                                            <Badge color="success">Domain</Badge>
                                        {/if}
                                        {#if checkerInfo.availability.limitToProviders && checkerInfo.availability.limitToProviders.length > 0}
                                            <Badge
                                                color="primary"
                                                title={checkerInfo.availability.limitToProviders.join(
                                                    ", ",
                                                )}
                                            >
                                                Provider-specific
                                            </Badge>
                                        {/if}
                                        {#if checkerInfo.availability.limitToServices && checkerInfo.availability.limitToServices.length > 0}
                                            <Badge
                                                color="info"
                                                title={checkerInfo.availability.limitToServices.join(
                                                    ", ",
                                                )}
                                            >
                                                Service-specific
                                            </Badge>
                                        {/if}
                                    {:else}
                                        <Badge color="secondary">General</Badge>
                                    {/if}
                                </td>
                                <td>
                                    <a
                                        href="/checkers/{checkerName}"
                                        class="btn btn-sm btn-primary"
                                    >
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
                Error loading checkers: {error.message}
            </p>
        </Card>
    {/await}
</Container>
