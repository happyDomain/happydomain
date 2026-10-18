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
    import { Badge, Card, CardBody, CardFooter, CardHeader, Icon, Table } from "@sveltestrap/sveltestrap";

    import type { SecretCounts } from "$lib/api-admin";
    import { sealedCount } from "./secrets";

    interface Props {
        objects: Record<string, SecretCounts>;
        class?: string;
    }

    let { objects, class: className = "" }: Props = $props();

    const types = $derived(Object.keys(objects).sort());

    const totals = $derived(
        types.reduce(
            (acc, t) => ({
                clear: acc.clear + (objects[t].clear ?? 0),
                sealed: acc.sealed + sealedCount(objects[t]),
                unreadable: acc.unreadable + (objects[t].unreadable ?? 0),
                undecodable: acc.undecodable + (objects[t].undecodable ?? 0),
            }),
            { clear: 0, sealed: 0, unreadable: 0, undecodable: 0 },
        ),
    );

    function sealedDetail(c: SecretCounts): string {
        return Object.entries(c.sealed ?? {})
            .map(([kind, n]) => `${kind}: ${n}`)
            .join(", ");
    }
</script>

{#snippet counters(clear: number, sealed: number, unreadable: number, undecodable: number, detail?: string)}
    <td class="text-end">{clear}</td>
    <td class="text-end" title={detail}>{sealed}</td>
    <td class="text-end">
        {#if unreadable > 0}
            <Badge color="danger">{unreadable}</Badge>
        {:else}
            0
        {/if}
    </td>
    <td class="text-end">
        {#if undecodable > 0}
            <Badge color="warning">{undecodable}</Badge>
        {:else}
            0
        {/if}
    </td>
{/snippet}

<Card class={className}>
    <CardHeader>
        <Icon name="table"></Icon>
        Stored credentials
    </CardHeader>
    <CardBody class="p-0">
        <div class="table-responsive">
            <Table hover class="mb-0">
                <thead>
                    <tr>
                        <th>Object type</th>
                        <th class="text-end">Clear</th>
                        <th class="text-end">Sealed</th>
                        <th class="text-end">Unreadable</th>
                        <th class="text-end">Undecodable</th>
                    </tr>
                </thead>
                <tbody>
                    {#if types.length === 0}
                        <tr>
                            <td colspan="5" class="text-center text-muted py-3">
                                No object type holds credentials
                            </td>
                        </tr>
                    {:else}
                        {#each types as t (t)}
                            <tr>
                                <td><code>{t}</code></td>
                                {@render counters(
                                    objects[t].clear ?? 0,
                                    sealedCount(objects[t]),
                                    objects[t].unreadable ?? 0,
                                    objects[t].undecodable ?? 0,
                                    sealedDetail(objects[t]),
                                )}
                            </tr>
                        {/each}
                    {/if}
                </tbody>
                {#if types.length > 1}
                    <tfoot>
                        <tr class="fw-bold">
                            <td>Total</td>
                            {@render counters(totals.clear, totals.sealed, totals.unreadable, totals.undecodable)}
                        </tr>
                    </tfoot>
                {/if}
            </Table>
        </div>
    </CardBody>
    <CardFooter class="small text-muted">
        <strong>Clear</strong>: stored in clear.
        <strong>Sealed</strong>: encrypted, and decrypting.
        <strong>Unreadable</strong>: encrypted, but no longer decrypting.
        <strong>Undecodable</strong>: objects that could not be looked at, such as a provider of a type this
        version no longer knows; they may hold encrypted credentials.
    </CardFooter>
</Card>
