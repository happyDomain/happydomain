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
    import { resolve } from "$app/paths";
    import { Button, Col, Container, Icon, Row, Table } from "@sveltestrap/sveltestrap";

    import { getUsersStats } from "$lib/api-admin";
    import type { ControllerUserStats } from "$lib/api-admin";

    const countsQ = getUsersStats();

    type SortField = "email" | "last_seen" | "provider_count" | "domain_count" | "zone_count";

    const columns: { field: SortField; label: string }[] = [
        { field: "email", label: "Email" },
        { field: "last_seen", label: "Last activity" },
        { field: "provider_count", label: "Providers" },
        { field: "domain_count", label: "Domains" },
        { field: "zone_count", label: "Zones" },
    ];

    const comparators: Record<
        SortField,
        (a: ControllerUserStats, b: ControllerUserStats) => number
    > = {
        email: (a, b) => (a.user?.email ?? "").localeCompare(b.user?.email ?? ""),
        last_seen: (a, b) =>
            new Date(a.user?.last_seen ?? "") > new Date(b.user?.last_seen ?? "") ? 1 : -1,
        provider_count: (a, b) => (a.provider_count ?? 0) - (b.provider_count ?? 0),
        domain_count: (a, b) => (a.domain_count ?? 0) - (b.domain_count ?? 0),
        zone_count: (a, b) => (a.zone_count ?? 0) - (b.zone_count ?? 0),
    };

    let sortBy = $state<SortField>("domain_count");
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
                User Counts
            </h1>
            <p class="text-muted lead">Resource usage per user account</p>
        </Col>
    </Row>

    {#await countsQ}
        Please wait...
    {:then countsR}
        {@const rows = countsR.data ?? []}
        {@const sorted = [...rows].sort((a, b) => (sortAsc ? 1 : -1) * comparators[sortBy](a, b))}
        <div class="table-responsive">
            <Table hover striped>
                <thead>
                    <tr>
                        {#each columns as col}
                            <th>
                                <Button
                                    color="link"
                                    class="p-0 text-decoration-none fw-bold"
                                    onclick={() => setSort(col.field)}
                                >
                                    {col.label}
                                    {#if sortBy === col.field}
                                        <Icon name={sortAsc ? "caret-up-fill" : "caret-down-fill"}
                                        ></Icon>
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
                                <a href={resolve("/users/[uid]", { uid: row.user?.id ?? "" })}
                                    >{row.user?.email}</a
                                >
                            </td>
                            <td>
                                {new Intl.DateTimeFormat(undefined, {
                                    dateStyle: "short",
                                    timeStyle: "medium",
                                }).format(
                                    new Date(
                                        row.user?.last_seen &&
                                            new Date(row.user?.last_seen).getFullYear() > 1
                                            ? row.user.last_seen
                                            : row.user?.created_at &&
                                                new Date(row.user?.created_at).getFullYear() > 1
                                              ? row.user?.created_at
                                              : null,
                                    ),
                                )}
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
        <p class="text-danger">Failed to load counts: {err}</p>
    {/await}
</Container>
