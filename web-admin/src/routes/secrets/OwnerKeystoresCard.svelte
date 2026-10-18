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
        Button,
        Card,
        CardBody,
        CardHeader,
        Icon,
        Input,
        InputGroup,
        Spinner,
        Table,
    } from "@sveltestrap/sveltestrap";

    import {
        deleteSecretsOwnersByOwnerKeystoresByRef,
        getSecretsOwnersByOwnerKeystores,
        postSecretsOwnersByOwnerKeystoresByRefMigrate,
        putSecretsOwnersByOwnerKeystoresDefault,
    } from "$lib/api-admin";
    import type { UsecaseKeystoreUsage } from "$lib/api-admin";
    import { toasts } from "$lib/stores/toasts";

    interface Props {
        class?: string;
    }

    let { class: className = "" }: Props = $props();

    // A user identifier, or "instance" for the instance itself, owner of the
    // options of the admin scope.
    let owner = $state("instance");
    let loadedOwner = $state("");
    let keystores = $state<UsecaseKeystoreUsage[] | null>(null);
    let loading = $state(false);
    let busy = $state<string | null>(null);
    let migrateTo = $state<Record<string, string>>({});

    function errorMessage(err: unknown): string {
        return (
            (err as { errmsg?: string; message?: string })?.errmsg ||
            (err as { message?: string })?.message ||
            String(err)
        );
    }

    async function load() {
        loading = true;
        try {
            const { data, error } = await getSecretsOwnersByOwnerKeystores({ path: { owner } });
            if (error) throw new Error(errorMessage(error));
            keystores = data ?? [];
            loadedOwner = owner;
        } catch (e) {
            keystores = null;
            toasts.addErrorToast({ message: errorMessage(e) });
        } finally {
            loading = false;
        }
    }

    async function act(
        ref: string,
        confirmText: string,
        call: () => Promise<{ data?: unknown; error?: unknown }>,
    ) {
        if (!confirm(confirmText)) return;
        busy = ref;
        try {
            const { data, error } = await call();
            if (error) throw new Error(errorMessage(error));
            const reports = Array.isArray(data)
                ? (data as { failed?: number; errors?: string[] }[])
                : [];
            const failed = reports.flatMap((r) => r.errors ?? []);
            if (failed.length > 0) {
                toasts.addErrorToast({
                    message: `Some objects could not move:\n${failed.join("\n")}`,
                });
            }
        } catch (e) {
            toasts.addErrorToast({ message: errorMessage(e) });
        } finally {
            busy = null;
            load();
        }
    }

    function label(ks: UsecaseKeystoreUsage): string {
        return ks.scope === "instance" ? `Instance (${ks.kind})` : `${ks.name} (${ks.kind})`;
    }
</script>

<Card class={className}>
    <CardHeader>
        <Icon name="safe"></Icon>
        Keystores of an owner
    </CardHeader>
    <CardBody>
        <form
            class="mb-3"
            onsubmit={(e) => {
                e.preventDefault();
                load();
            }}
        >
            <InputGroup size="sm">
                <Input
                    type="text"
                    bind:value={owner}
                    placeholder="User identifier, or instance"
                    aria-label="Owner"
                />
                <Button type="submit" color="primary" disabled={loading || !owner}>
                    {#if loading}<Spinner size="sm" />{:else}<Icon name="search"></Icon>{/if}
                    Load
                </Button>
            </InputGroup>
            <small class="text-muted">
                A user identifier, or <code>instance</code> for the options of the admin scope.
            </small>
        </form>

        {#if keystores}
            <div class="table-responsive">
                <Table hover size="sm" class="mb-0">
                    <thead>
                        <tr>
                            <th>Keystore</th>
                            <th>Holds</th>
                            <th class="text-end">Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each keystores as ks (ks.ref)}
                            {@const objects = ks.objects ?? []}
                            {@const targets = keystores.filter(
                                (o) => o.ref !== ks.ref && o.available,
                            )}
                            <tr>
                                <td>
                                    {label(ks)}
                                    {#if ks.default}<Badge color="primary" class="ms-1"
                                            >default</Badge
                                        >{/if}
                                    {#if !ks.unattended}<Badge color="warning" class="ms-1"
                                            >needs its owner</Badge
                                        >{/if}
                                    {#if !ks.available}<Badge color="danger" class="ms-1"
                                            >store not configured</Badge
                                        >{/if}
                                    <small class="d-block text-muted"><code>{ks.ref}</code></small>
                                </td>
                                <td class="small">
                                    {#if objects.length === 0}
                                        <span class="text-muted">nothing</span>
                                    {:else}
                                        {#each objects as o (`${o.objectType}|${o.objectId}`)}
                                            <div>{o.objectType}: {o.label || o.objectId}</div>
                                        {/each}
                                    {/if}
                                </td>
                                <td class="text-end">
                                    <div class="d-flex flex-wrap gap-1 justify-content-end">
                                        {#if !ks.default && ks.available}
                                            <Button
                                                size="sm"
                                                color="primary"
                                                outline
                                                disabled={busy !== null}
                                                onclick={() =>
                                                    act(
                                                        ks.ref!,
                                                        `Make ${label(ks)} the default keystore of ${loadedOwner}?`,
                                                        () =>
                                                            putSecretsOwnersByOwnerKeystoresDefault(
                                                                {
                                                                    path: { owner: loadedOwner },
                                                                    body: { ref: ks.ref },
                                                                },
                                                            ),
                                                    )}
                                            >
                                                Default
                                            </Button>
                                        {/if}
                                        {#if objects.length > 0 && targets.length > 0}
                                            <InputGroup size="sm" class="w-auto">
                                                <Input
                                                    type="select"
                                                    bsSize="sm"
                                                    bind:value={migrateTo[ks.ref!]}
                                                    aria-label="Migrate to"
                                                >
                                                    <option value="">Migrate to...</option>
                                                    {#each targets as target (target.ref)}
                                                        <option value={target.ref}
                                                            >{label(target)}</option
                                                        >
                                                    {/each}
                                                </Input>
                                                <Button
                                                    color="secondary"
                                                    outline
                                                    disabled={busy !== null || !migrateTo[ks.ref!]}
                                                    onclick={() =>
                                                        act(
                                                            ks.ref!,
                                                            `Move what ${label(ks)} holds into the chosen keystore?`,
                                                            () =>
                                                                postSecretsOwnersByOwnerKeystoresByRefMigrate(
                                                                    {
                                                                        path: {
                                                                            owner: loadedOwner,
                                                                            ref: ks.ref!,
                                                                        },
                                                                        body: {
                                                                            to: migrateTo[ks.ref!],
                                                                        },
                                                                    },
                                                                ),
                                                        )}
                                                >
                                                    Migrate
                                                </Button>
                                            </InputGroup>
                                        {/if}
                                        {#if ks.safeId}
                                            <Button
                                                size="sm"
                                                color="danger"
                                                outline
                                                title={objects.length > 0
                                                    ? "Migrate what it holds first"
                                                    : "Delete"}
                                                disabled={busy !== null || objects.length > 0}
                                                onclick={() =>
                                                    act(
                                                        ks.ref!,
                                                        `Delete ${label(ks)} of ${loadedOwner}? Whatever it still holds would no longer open.`,
                                                        () =>
                                                            deleteSecretsOwnersByOwnerKeystoresByRef(
                                                                {
                                                                    path: {
                                                                        owner: loadedOwner,
                                                                        ref: ks.ref!,
                                                                    },
                                                                },
                                                            ),
                                                    )}
                                            >
                                                {#if busy === ks.ref}<Spinner
                                                        size="sm"
                                                    />{:else}<Icon name="trash"></Icon>{/if}
                                            </Button>
                                        {/if}
                                    </div>
                                </td>
                            </tr>
                        {/each}
                    </tbody>
                </Table>
            </div>
        {/if}
    </CardBody>
</Card>
