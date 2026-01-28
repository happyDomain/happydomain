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

    import { t } from "$lib/translations";
    import { listChecks } from "$lib/api/checks";

    let checksPromise = $state(listChecks());

    let searchQuery = $state("");
</script>

<svelte:head>
    <title>{$t("checkers.title")} - happyDomain</title>
</svelte:head>

<Container class="flex-fill my-5">
    <Row class="mb-4">
        <Col md={8}>
            <h1 class="display-5">
                <Icon name="check-circle-fill"></Icon>
                {$t("checkers.title")}
            </h1>
            <p class="d-flex gap-3 align-items-center text-muted">
                <span class="lead">
                    {$t("checkers.description")}
                </span>
                {#await checksPromise then checkers}
                    <span
                        >{$t("checkers.available-count", {
                            count: Object.keys(checkers ?? {}).length,
                        })}</span
                    >
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
                    placeholder={$t("checkers.search-placeholder")}
                    bind:value={searchQuery}
                />
            </InputGroup>
        </Col>
    </Row>

    {#await checksPromise}
        <Card body>
            <p class="text-center mb-0">
                <span class="spinner-border spinner-border-sm me-2"></span>
                {$t("checkers.loading")}
            </p>
        </Card>
    {:then checks}
        <div class="table-responsive">
            <Table hover bordered>
                <thead>
                    <tr>
                        <th>{$t("checkers.table.name")}</th>
                        <th>{$t("checkers.table.availability")}</th>
                        <th>{$t("checkers.table.actions")}</th>
                    </tr>
                </thead>
                <tbody>
                    {#if !checks || Object.keys(checks).length == 0}
                        <tr>
                            <td colspan="4" class="text-center text-muted py-4">
                                {$t("checkers.no-checkers")}
                            </td>
                        </tr>
                    {:else}
                        {#each Object.entries(checks ?? {}).filter(([name, _info]) => name
                                    .toLowerCase()
                                    .indexOf(searchQuery.toLowerCase()) > -1) as [checkerName, checkerInfo]}
                            <tr>
                                <td><strong>{checkerInfo.name || checkerName}</strong></td>
                                <td>
                                    {#if checkerInfo.availableOn}
                                        {#if checkerInfo.availableOn.applyToDomain}
                                            <Badge color="success">
                                                {$t("checkers.availability.domain")}
                                            </Badge>
                                        {/if}
                                        {#if checkerInfo.availableOn.limitToProviders && checkerInfo.availableOn.limitToProviders.length > 0}
                                            <Badge
                                                color="primary"
                                                title={checkerInfo.availableOn.limitToProviders.join(
                                                    ", ",
                                                )}
                                            >
                                                {$t("checkers.availability.provider-specific")}
                                            </Badge>
                                        {/if}
                                        {#if checkerInfo.availableOn.limitToServices && checkerInfo.availableOn.limitToServices.length > 0}
                                            <Badge
                                                color="info"
                                                title={checkerInfo.availableOn.limitToServices.join(
                                                    ", ",
                                                )}
                                            >
                                                {$t("checkers.availability.service-specific")}
                                            </Badge>
                                        {/if}
                                    {:else}
                                        <Badge color="secondary">
                                            {$t("checkers.availability.general")}
                                        </Badge>
                                    {/if}
                                </td>
                                <td>
                                    <a
                                        href="/checkers/{checkerName}"
                                        class="btn btn-sm btn-primary"
                                    >
                                        <Icon name="gear-fill"></Icon>
                                        {$t("checkers.actions.configure")}
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
                {$t("checkers.error-loading", { error: error.message })}
            </p>
        </Card>
    {/await}
</Container>
