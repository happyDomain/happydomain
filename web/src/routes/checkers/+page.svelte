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
        Row,
    } from "@sveltestrap/sveltestrap";

    import PageTitle from "$lib/components/PageTitle.svelte";
    import CheckersAvailabilityTable from "$lib/components/checkers/CheckersAvailabilityTable.svelte";
    import { t } from "$lib/translations";
    import { checkers } from "$lib/stores/checkers";

    let searchQuery = $state("");

    let filteredCheckers = $derived(
        $checkers
            ? Object.entries($checkers).filter(([name]) =>
                  name.toLowerCase().includes(searchQuery.toLowerCase()),
              )
            : [],
    );
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
        {#if Object.keys($checkers).length == 0}
            <p class="text-center text-muted py-4">
                {$t("checkers.no-checkers")}
            </p>
        {:else}
            <CheckersAvailabilityTable
                checkers={filteredCheckers}
                basePath="/checkers"
            />
        {/if}
    {/if}
</Container>
