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
    import { Badge, Icon, Table } from "@sveltestrap/sveltestrap";

    import { navigate } from "$lib/stores/config";
    import { t } from "$lib/translations";
    import type { CheckerInfo } from "$lib/model/checker";

    interface Props {
        checkers: [string, CheckerInfo][];
        basePath: string;
        configureKey?: string;
    }

    let {
        checkers,
        basePath,
        configureKey = "checkers.actions.configure",
    }: Props = $props();
</script>

<Table striped hover responsive>
    <thead>
        <tr>
            <th>{$t("checkers.table.name")}</th>
            <th>{$t("checkers.table.availability")}</th>
            <th></th>
        </tr>
    </thead>
    <tbody>
        {#each checkers as [checkerName, checkerInfo]}
            <tr
                style="cursor: pointer"
                onclick={() => navigate(`${basePath}/${encodeURIComponent(checkerName)}`)}
            >
                <td><strong>{checkerInfo.name || checkerName}</strong></td>
                <td>
                    {#if checkerInfo.availability}
                        {#if checkerInfo.availability.applyToDomain}
                            <Badge color="success">
                                {$t("checkers.availability.domain")}
                            </Badge>
                        {/if}
                        {#if checkerInfo.availability.applyToZone}
                            <Badge color="success">
                                {$t("checkers.availability.zone")}
                            </Badge>
                        {/if}
                        {#if checkerInfo.availability.limitToProviders && checkerInfo.availability.limitToProviders.length > 0}
                            <Badge
                                color="primary"
                                title={checkerInfo.availability.limitToProviders.join(", ")}
                            >
                                {$t("checkers.availability.provider-specific")}
                            </Badge>
                        {/if}
                        {#if checkerInfo.availability.limitToServices && checkerInfo.availability.limitToServices.length > 0}
                            <Badge
                                color="info"
                                title={checkerInfo.availability.limitToServices.join(", ")}
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
                        href="{basePath}/{encodeURIComponent(checkerName)}"
                        class="btn btn-sm btn-outline-primary"
                    >
                        <Icon name="gear-fill"></Icon>
                        {$t(configureKey)}
                    </a>
                </td>
            </tr>
        {/each}
    </tbody>
</Table>
