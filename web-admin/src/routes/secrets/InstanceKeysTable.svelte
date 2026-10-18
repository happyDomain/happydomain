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
        Badge,
        Card,
        CardBody,
        CardFooter,
        CardHeader,
        Icon,
        Table,
    } from "@sveltestrap/sveltestrap";

    import type { SecretKeyStatus } from "$lib/api-admin";

    interface Props {
        keys: SecretKeyStatus[];
        class?: string;
    }

    let { keys, class: className = "" }: Props = $props();

    // MISSING and UNREADABLE are not keys of the keyset: they count the safes
    // that no key of the keyset opens.
    function isBroken(k: SecretKeyStatus): boolean {
        return k.status === "MISSING" || k.status === "UNREADABLE";
    }

    function statusColor(k: SecretKeyStatus): string {
        if (isBroken(k)) return "danger";
        if (k.status === "ENABLED") return "success";
        return "secondary";
    }
</script>

<Card class={className}>
    <CardHeader>
        <Icon name="key-fill"></Icon>
        Instance keys
        <Badge color="secondary" class="ms-2">{keys.filter((k) => !isBroken(k)).length}</Badge>
    </CardHeader>
    <CardBody class="p-0">
        <div class="table-responsive">
            <Table hover class="mb-0">
                <thead>
                    <tr>
                        <th>Id</th>
                        <th>Status</th>
                        <th>Type</th>
                        <th class="text-end">Users' keys</th>
                    </tr>
                </thead>
                <tbody>
                    {#if keys.length === 0}
                        <tr>
                            <td colspan="4" class="text-center text-muted py-3">
                                No instance keyset configured
                            </td>
                        </tr>
                    {:else}
                        {#each keys as k, i (i)}
                            <tr class:table-danger={isBroken(k)}>
                                <td>
                                    {#if k.status === "UNREADABLE"}
                                        <span class="text-muted">-</span>
                                    {:else}
                                        <code>{k.id}</code>
                                    {/if}
                                    {#if k.primary}
                                        <Badge color="primary" class="ms-1">primary</Badge>
                                    {/if}
                                </td>
                                <td><Badge color={statusColor(k)}>{k.status}</Badge></td>
                                <td class="small text-muted">{k.type ?? ""}</td>
                                <td class="text-end">
                                    {k.safes ?? 0}
                                    {#if !k.primary && !isBroken(k) && !k.safes}
                                        <small class="d-block text-muted">
                                            can be removed (<code
                                                >happydomain secret-keyset delete-key &lt;file&gt; {k.id}</code
                                            >)
                                        </small>
                                    {/if}
                                </td>
                            </tr>
                        {/each}
                    {/if}
                </tbody>
            </Table>
        </div>
    </CardBody>
    {#if keys.some(isBroken)}
        <CardFooter class="small text-danger">
            <Icon name="exclamation-triangle-fill"></Icon>
            <strong>MISSING</strong>: users' keys protected by a key removed from the keyset; put
            back a copy of the keyset holding it.
            <strong>UNREADABLE</strong>: users' keys too damaged to tell which key protects them.
        </CardFooter>
    {/if}
</Card>
