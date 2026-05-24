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
        Col,
        Container,
        Icon,
        Row,
        Table,
    } from "@sveltestrap/sveltestrap";

    import { getUsersStats } from '$lib/api-admin';
    import type { ControllerUserStats } from '$lib/api-admin';

    const statsQ = getUsersStats();

    type SortField = 'email' | 'provider_count' | 'domain_count' | 'zone_count';

    const columns: { field: SortField; label: string }[] = [
        { field: 'email', label: 'Email' },
        { field: 'provider_count', label: 'Providers' },
        { field: 'domain_count', label: 'Domains' },
        { field: 'zone_count', label: 'Zones' },
    ];

    const comparators: Record<SortField, (a: ControllerUserStats, b: ControllerUserStats) => number> = {
        email:          (a, b) => (a.user?.email ?? '').localeCompare(b.user?.email ?? ''),
        provider_count: (a, b) => (a.provider_count ?? 0) - (b.provider_count ?? 0),
        domain_count:   (a, b) => (a.domain_count ?? 0) - (b.domain_count ?? 0),
        zone_count:     (a, b) => (a.zone_count ?? 0) - (b.zone_count ?? 0),
    };

    let sortBy = $state<SortField>('domain_count');
    let sortAsc = $state(false);

    function setSort(field: SortField) {
        if (sortBy === field) {
            sortAsc = !sortAsc;
        } else {
            sortBy = field;
            sortAsc = true;
        }
    }
</script>

<Container class="flex-fill my-5">
    <Row class="mb-4">
        <Col>
            <h1 class="display-5">
                <Icon name="bar-chart-fill"></Icon>
                User Statistics
            </h1>
            <p class="text-muted lead">
                Resource usage per user account
            </p>
        </Col>
    </Row>

    {#await statsQ}
        Please wait...
    {:then statsR}
        {@const rows = statsR.data ?? []}
        {@const sorted = [...rows].sort((a, b) => (sortAsc ? 1 : -1) * comparators[sortBy](a, b))}
        <div class="table-responsive">
            <Table hover bordered>
                <thead>
                    <tr>
                        {#each columns as col}
                            <th>
                                <Button color="link" class="p-0 text-decoration-none fw-bold" onclick={() => setSort(col.field)}>
                                    {col.label}
                                    {#if sortBy === col.field}
                                        <Icon name={sortAsc ? 'caret-up-fill' : 'caret-down-fill'}></Icon>
                                    {:else}
                                        <Icon name="caret-up" class="text-muted"></Icon>
                                    {/if}
                                </Button>
                            </th>
                        {/each}
                    </tr>
                </thead>
                <tbody>
                    {#each sorted as row}
                        <tr>
                            <td>
                                <a href="/users/{row.user?.id}">{row.user?.email}</a>
                            </td>
                            <td>{row.provider_count ?? 0}</td>
                            <td>{row.domain_count ?? 0}</td>
                            <td>{row.zone_count ?? 0}</td>
                        </tr>
                    {/each}
                </tbody>
            </Table>
        </div>
    {:catch err}
        <p class="text-danger">Failed to load statistics: {err}</p>
    {/await}
</Container>
