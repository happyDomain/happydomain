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
    import { Badge, Table } from "@sveltestrap/sveltestrap";
    import type { CheckerCheckerDefinition } from "$lib/api-base/types.gen";
    import { t } from "$lib/translations";
    import { availabilityBadges } from "$lib/utils";

    let {
        checkers,
        basePath = "/checkers",
    }: {
        checkers: [string, CheckerCheckerDefinition][];
        basePath?: string;
    } = $props();
</script>

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
            {#each checkers as [checkerId, checkerInfo]}
                {@const badges = availabilityBadges(checkerInfo.availability, $t)}
                <tr>
                    <td><strong>{checkerInfo.name || checkerId}</strong></td>
                    <td>
                        {#if badges.length > 0}
                            <div class="d-flex flex-wrap gap-1">
                                {#each badges as badge}
                                    <Badge color={badge.color}>
                                        {badge.label}
                                    </Badge>
                                {/each}
                            </div>
                        {:else}
                            <Badge color="secondary">{$t("checkers.availability.general")}</Badge>
                        {/if}
                    </td>
                    <td>
                        <a href="{basePath}/{checkerId}" class="btn btn-sm btn-primary">
                            {$t("checkers.table.manage")}
                        </a>
                    </td>
                </tr>
            {/each}
        </tbody>
    </Table>
</div>
