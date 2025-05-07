<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2024 happyDomain
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
    import { Badge, Button, Icon, Table, Spinner } from "@sveltestrap/sveltestrap";

    import { getDomainLogs } from "$lib/api/domains";
    import type { Domain } from "$lib/model/domain";
    import { getUser } from "$lib/stores/users";
    import { t } from "$lib/translations";

    export let data: { domain: Domain; history: string };
</script>

<div class="flex-fill pb-4 pt-2">
    <h2>{$t("logs.title")} <span class="font-monospace">{data.domain.domain}</span></h2>
    {#await getDomainLogs(data.domain.id)}
        <div class="mt-5 text-center flex-fill">
            <Spinner label="Spinning" />
            <p>{$t("wait.loading")}</p>
        </div>
    {:then logs}
        <Table hover striped>
            <thead>
                <tr>
                    <th>{$t("logs.user")}</th>
                    <th>{$t("logs.level")}</th>
                    <th>{$t("logs.description")}</th>
                    <th>{$t("logs.date")}</th>
                </tr>
            </thead>
            <tbody>
                {#if logs}
                    {#each logs as log}
                        <tr>
                            <td class="align-middle" style="max-width: 150px">
                                <div class="d-flex align-items-center gap-1">
                                    {#await getUser(log.id_user)}
                                        <img
                                            src={"/api/users/" +
                                                encodeURIComponent(log.id_user) +
                                                "/avatar.png"}
                                            alt={log.id_user}
                                            style="height: 1.1em; border-radius: .1em"
                                        />
                                        {log.id_user}
                                    {:then user}
                                        <img
                                            src={"/api/users/" +
                                                encodeURIComponent(log.id_user) +
                                                "/avatar.png"}
                                            alt={user.email}
                                            style="height: 1.1em; border-radius: .1em"
                                        />
                                        <span title={user.email} class="text-truncate">
                                            {user.email}
                                        </span>
                                    {/await}
                                </div>
                            </td>
                            <td class="align-middle text-center">
                                {#if log.level > 9}<Badge color="light">DEBUG</Badge>
                                {:else if log.level > 8}<Badge color="success">INFO</Badge>
                                {:else if log.level > 7}<Badge color="info">INFO</Badge>
                                {:else if log.level > 3}<Badge color="warning">WARN</Badge>
                                {:else if log.level > 1}<Badge color="danger">ERR</Badge>
                                {:else}<Badge color="dark">CRIT</Badge>
                                {/if}
                            </td>
                            <td class="align-middle">
                                {log.content}
                            </td>
                            <td>
                                {new Intl.DateTimeFormat(undefined, {
                                    dateStyle: "short",
                                    timeStyle: "medium",
                                }).format(new Date(log.date))}
                            </td>
                        </tr>
                    {/each}
                {:else}
                    <tr>
                        <td colspan="4" class="text-center">
                            {$t("logs.no-entry")}
                        </td>
                    </tr>
                {/if}
            </tbody>
        </Table>
    {/await}
</div>
