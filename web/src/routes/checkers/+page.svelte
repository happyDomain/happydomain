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

    import PageTitle from "$lib/components/PageTitle.svelte";
    import { navigate } from "$lib/stores/config";
    import { t } from "$lib/translations";
    import { checkers } from "$lib/stores/checkers";

    let searchQuery = $state("");
</script>

<svelte:head>
    <title>{$t("checkers.title")} - happyDomain</title>
</svelte:head>

<Container class="flex-fill my-5">
    <PageTitle title={$t("checkers.title")} subtitle={$t("checkers.description")}>
        {#if $checkers}
            {$t("checkers.available-count", {
                count: Object.keys($checkers).length,
            })}
        {/if}
    </PageTitle>

    <Row class="mb-4 mt-3">
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

    {#if !$checkers}
        <Card body>
            <p class="text-center mb-0">
                <span class="spinner-border spinner-border-sm me-2"></span>
                {$t("checkers.loading")}
            </p>
        </Card>
    {:else}
        <Table striped hover responsive>
            <thead>
                <tr>
                    <th>{$t("checkers.table.name")}</th>
                    <th>{$t("checkers.table.availability")}</th>
                    <th></th>
                </tr>
            </thead>
            <tbody>
                {#if Object.keys($checkers).length == 0}
                    <tr>
                        <td colspan="4" class="text-center text-muted py-4">
                            {$t("checkers.no-checkers")}
                        </td>
                    </tr>
                {:else}
                    {#each Object.entries($checkers).filter(([name, _info]) => name
                                .toLowerCase()
                                .indexOf(searchQuery.toLowerCase()) > -1) as [checkerName, checkerInfo]}
                        <tr
                            style="cursor: pointer"
                            onclick={() => navigate("/checkers/" + checkerName)}
                        >
                            <td><strong>{checkerInfo.name || checkerName}</strong></td>
                            <td>
                                {#if checkerInfo.availability}
                                    {#if checkerInfo.availability.applyToDomain}
                                        <Badge color="success">
                                            {$t("checkers.availability.domain")}
                                        </Badge>
                                    {/if}
                                    {#if checkerInfo.availability.limitToProviders && checkerInfo.availability.limitToProviders.length > 0}
                                        <Badge
                                            color="primary"
                                            title={checkerInfo.availability.limitToProviders.join(
                                                ", ",
                                            )}
                                        >
                                            {$t("checkers.availability.provider-specific")}
                                        </Badge>
                                    {/if}
                                    {#if checkerInfo.availability.limitToServices && checkerInfo.availability.limitToServices.length > 0}
                                        <Badge
                                            color="info"
                                            title={checkerInfo.availability.limitToServices.join(
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
                            <td class="text-end">
                                <a
                                    href="/checkers/{checkerName}"
                                    class="btn btn-sm btn-outline-primary"
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
    {/if}
</Container>
