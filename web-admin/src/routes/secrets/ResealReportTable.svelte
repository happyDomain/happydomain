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

    import type { SecretResealReport } from "$lib/api-admin";

    interface Props {
        reports: SecretResealReport[];
    }

    let { reports }: Props = $props();
</script>

<div class="table-responsive">
    <Table size="sm" class="mb-0">
        <thead>
            <tr>
                <th>Object type</th>
                <th class="text-end">Processed</th>
                <th class="text-end">Changed</th>
                <th class="text-end">Failed</th>
            </tr>
        </thead>
        <tbody>
            {#each reports as r, i (i)}
                <tr>
                    <td><code>{r.objectType}</code></td>
                    <td class="text-end">{r.processed ?? 0}</td>
                    <td class="text-end">{r.changed ?? 0}</td>
                    <td class="text-end">
                        {#if r.failed}
                            <Badge color="danger">{r.failed}</Badge>
                        {:else}
                            0
                        {/if}
                    </td>
                </tr>
                {#if r.errors?.length}
                    <tr>
                        <td colspan="4">
                            <details>
                                <summary class="text-danger small">
                                    {r.errors.length} error{r.errors.length > 1 ? "s" : ""}
                                </summary>
                                <ul class="small mb-0">
                                    {#each r.errors as e, j (j)}
                                        <li><code>{e}</code></li>
                                    {/each}
                                </ul>
                            </details>
                        </td>
                    </tr>
                {/if}
            {/each}
        </tbody>
    </Table>
</div>
