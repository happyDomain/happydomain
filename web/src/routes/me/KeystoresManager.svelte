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
        Input,
        InputGroup,
        ListGroup,
        ListGroupItem,
        ModalBody,
        ModalFooter,
        ModalHeader,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import Modal from "$lib/components/Modal.svelte";
    import {
        createKeystore,
        deleteKeystore,
        listKeystoreKinds,
        lockKeystore,
        migrateKeystore,
        setDefaultKeystore,
        updateKeystore,
        type Keystore,
        type KeystoreKind,
    } from "$lib/api/keystores";
    import ResourceInput from "$lib/components/inputs/basic.svelte";
    import type { Field } from "$lib/model/custom_form.svelte";
    import { appConfig } from "$lib/stores/config";
    import { keystoreKindLabel, keystores, refreshKeystores } from "$lib/stores/keystores";
    import { toasts } from "$lib/stores/toasts";
    import { requestUnlock } from "$lib/stores/unlock";
    import { t } from "$lib/translations";

    let keystores_req = $state(refreshKeystores());
    let kinds_req = listKeystoreKinds();

    let busy = $state(new Set<string>());
    let migrateTo: Record<string, string> = $state({});

    function label(ks: Keystore): string {
        return ks.scope === "instance"
            ? $t("keystores.instance", { kind: ks.kind ?? "" })
            : (ks.name ?? "");
    }

    async function act(ref: string, fn: () => Promise<unknown>) {
        busy.add(ref);
        busy = busy;
        try {
            await fn();
        } catch (err) {
            toasts.addErrorToast({ message: String(err), timeout: 10000 });
        } finally {
            busy.delete(ref);
            busy = busy;
            keystores_req = refreshKeystores();
        }
    }

    function makeDefault(ks: Keystore) {
        act(ks.ref!, () => setDefaultKeystore(ks.ref!));
    }

    function unlock(ks: Keystore) {
        act(ks.ref!, () =>
            requestUnlock({ ref: ks.ref, kind: ks.kind, name: ks.name, form: ks.unlockForm }),
        );
    }

    function lock(ks: Keystore) {
        act(ks.ref!, () => lockKeystore(ks.ref!));
    }

    // The keystore whose update form is open, and what is entered in it.
    let updating = $state("");
    let updateParams: Record<string, unknown> = $state({});

    function toggleUpdate(ks: Keystore) {
        updating = updating === ks.ref ? "" : ks.ref!;
        updateParams = {};
    }

    function update(ks: Keystore) {
        act(ks.ref!, async () => {
            await updateKeystore(ks.ref!, updateParams);
            updating = "";
            updateParams = {};
            toasts.addToast({ message: $t("keystores.updated"), type: "success", timeout: 5000 });
        });
    }

    function del(ks: Keystore) {
        act(ks.ref!, () => deleteKeystore(ks.ref!));
    }

    function migrate(ks: Keystore) {
        const to = migrateTo[ks.ref!];
        if (!to) return;
        act(ks.ref!, async () => {
            const reports = await migrateKeystore(ks.ref!, to);
            const changed = reports.reduce((n, r) => n + (r.changed ?? 0), 0);
            const failed = reports.reduce((n, r) => n + (r.failed ?? 0), 0);
            const errors = reports.flatMap((r) => r.errors ?? []);
            if (failed > 0) {
                toasts.addErrorToast({
                    message:
                        $t("keystores.migrated-partly", { changed, failed }) +
                        "\n" +
                        errors.join("\n"),
                    timeout: 15000,
                });
            } else {
                toasts.addToast({
                    message: $t("keystores.migrated", { changed }),
                    type: "success",
                    timeout: 5000,
                });
            }
        });
    }

    let createModalOpen = $state(false);
    let newKind = $state("");
    let newName = $state("");
    let newParams: Record<string, unknown> = $state({});
    let creating = $state(false);

    function openCreate() {
        newName = "";
        newParams = {};
        createModalOpen = true;
    }

    async function create(kinds: Array<KeystoreKind>) {
        const kind = newKind || kinds[0]?.kind;
        if (!kind || !newName) return;
        creating = true;
        try {
            await createKeystore(kind, newName, newParams);
            newName = "";
            newParams = {};
            createModalOpen = false;
            keystores_req = refreshKeystores();
        } catch (err) {
            toasts.addErrorToast({ message: String(err), timeout: 10000 });
        } finally {
            creating = false;
        }
    }
</script>

{#await kinds_req then kinds}
    {#if kinds.length > 0}
        <div class="d-flex justify-content-end mb-2">
            <Button color="info" outline on:click={openCreate}>
                <i class="bi bi-plus-lg"></i>
                {$t("keystores.create")}
            </Button>
        </div>
    {/if}
{/await}

{#await keystores_req}
    <div class="d-flex justify-content-center my-2">
        <Spinner />
    </div>
{:then}
    <ListGroup>
        {#each $keystores ?? [] as ks (ks.ref)}
            {@const objects = ks.objects ?? []}
            <ListGroupItem class="d-flex flex-column flex-md-row gap-2 justify-content-between">
                <div class="flex-fill">
                    <div class="fw-bold">
                        <i class="bi bi-safe"></i>
                        {label(ks)}
                        {#if ks.default}
                            <Badge color="primary">{$t("keystores.default")}</Badge>
                        {/if}
                        <Badge color="secondary">
                            {$t(
                                ks.scope === "instance"
                                    ? "keystores.scope-instance"
                                    : "keystores.scope-personal",
                            )}
                        </Badge>
                        {#if !ks.unattended}
                            <Badge color="warning" title={$t("keystores.needs-owner-help")}>
                                {$t("keystores.needs-owner")}
                            </Badge>
                        {/if}
                        {#if ks.interactive}
                            <Badge color={ks.locked ? "dark" : "success"}>
                                <i class={ks.locked ? "bi bi-lock" : "bi bi-unlock"}></i>
                                {$t(ks.locked ? "keystores.locked" : "keystores.unlocked")}
                            </Badge>
                        {/if}
                        {#if ks.references}
                            <Badge color="info" title={ks.referenceHint}>
                                <i class="bi bi-link-45deg"></i>
                                {$t("keystores.references")}
                            </Badge>
                        {/if}
                        {#if !ks.available}
                            <Badge color="danger">{$t("keystores.unavailable")}</Badge>
                        {/if}
                    </div>
                    <div class="small">
                        <span class="text-muted">{$t("keystores.kind")}</span>
                        {keystoreKindLabel($t, ks.kind ?? "")}
                        <code class="text-muted">{ks.kind}</code>
                    </div>
                    <div class="text-muted small">
                        {#if objects.length === 0}
                            {$t("keystores.unused")}
                        {:else}
                            {$t("keystores.used-by", { n: objects.length })}
                            {objects.map((o) => o.label || o.objectId).join(", ")}
                        {/if}
                    </div>
                    {#if updating === ks.ref}
                        <form
                            class="mt-2"
                            onsubmit={(e) => {
                                e.preventDefault();
                                update(ks);
                            }}
                        >
                            {#each ks.updateForm ?? [] as field (field.id)}
                                <ResourceInput
                                    edit
                                    index={"keystore-update-" + field.id}
                                    specs={field as Field}
                                    bind:value={updateParams[field.id!]}
                                />
                            {/each}
                            <Button
                                type="submit"
                                color="primary"
                                size="sm"
                                disabled={busy.has(ks.ref!)}
                            >
                                {#if busy.has(ks.ref!)}
                                    <Spinner size="sm" />
                                {/if}
                                {$t("keystores.update")}
                            </Button>
                            <Button
                                color="secondary"
                                outline
                                size="sm"
                                on:click={() => toggleUpdate(ks)}
                            >
                                {$t("common.cancel")}
                            </Button>
                        </form>
                    {/if}
                </div>
                <div class="d-flex flex-wrap gap-1 align-items-start">
                    {#if ks.interactive && ks.available}
                        <Button
                            color="secondary"
                            outline
                            size="sm"
                            disabled={busy.has(ks.ref!)}
                            on:click={() => (ks.locked ? unlock(ks) : lock(ks))}
                        >
                            <i class={ks.locked ? "bi bi-unlock" : "bi bi-lock"}></i>
                            {$t(ks.locked ? "keystores.unlock" : "keystores.lock")}
                        </Button>
                    {/if}
                    {#if ks.updateForm?.length && ks.available && updating !== ks.ref}
                        <Button
                            color="secondary"
                            outline
                            size="sm"
                            disabled={busy.has(ks.ref!)}
                            on:click={() => toggleUpdate(ks)}
                        >
                            <i class="bi bi-key"></i>
                            {$t("keystores.update")}
                        </Button>
                    {/if}
                    {#if !ks.default && ks.available}
                        <Button
                            color="primary"
                            outline
                            size="sm"
                            disabled={busy.has(ks.ref!)}
                            on:click={() => makeDefault(ks)}
                        >
                            {$t("keystores.set-default")}
                        </Button>
                    {/if}
                    <!-- A keystore of references keeps secrets out of happyDomain: leaving it is opt-in. -->
                    {#if objects.length > 0 && (!ks.references || $appConfig.enable_keystore_extraction)}
                        <!-- A keystore of references would take the secrets for references. -->
                        {@const targets = ($keystores ?? []).filter(
                            (o) => o.ref !== ks.ref && o.available && !o.references,
                        )}
                        {#if targets.length > 0}
                            <InputGroup size="sm" class="w-auto">
                                <Input
                                    type="select"
                                    bsSize="sm"
                                    aria-label={$t("keystores.migrate-to")}
                                    bind:value={migrateTo[ks.ref!]}
                                >
                                    <option value="">{$t("keystores.migrate-to")}</option>
                                    {#each targets as target (target.ref)}
                                        <option value={target.ref}>{label(target)}</option>
                                    {/each}
                                </Input>
                                <Button
                                    color="secondary"
                                    outline
                                    disabled={!migrateTo[ks.ref!] || busy.has(ks.ref!)}
                                    on:click={() => migrate(ks)}
                                >
                                    {$t("keystores.migrate")}
                                </Button>
                            </InputGroup>
                        {/if}
                    {/if}
                    {#if ks.scope === "personal"}
                        <Button
                            color="danger"
                            outline
                            size="sm"
                            title={objects.length > 0
                                ? $t("keystores.delete-used")
                                : $t("common.delete")}
                            disabled={objects.length > 0 || busy.has(ks.ref!)}
                            on:click={() => del(ks)}
                        >
                            {#if busy.has(ks.ref!)}
                                <Spinner size="sm" />
                            {:else}
                                <i class="bi bi-trash"></i>
                            {/if}
                        </Button>
                    {/if}
                </div>
            </ListGroupItem>
        {/each}
    </ListGroup>
{/await}

{#await kinds_req then kinds}
    {#if kinds.length > 0}
        {@const kind = kinds.find((k) => k.kind === (newKind || kinds[0].kind))}
        <Modal isOpen={createModalOpen} toggle={() => (createModalOpen = !createModalOpen)}>
            <ModalHeader toggle={() => (createModalOpen = !createModalOpen)}>
                {$t("keystores.create")}
            </ModalHeader>
            <ModalBody>
                <form
                    id="keystore-create-form"
                    onsubmit={(e) => {
                        e.preventDefault();
                        create(kinds);
                    }}
                >
                    <div class="mb-2">
                        <label for="keystore-kind">{$t("keystores.kind")}</label>
                        <Input id="keystore-kind" type="select" bind:value={newKind}>
                            {#each kinds as k (k.kind)}
                                <option value={k.kind}>{keystoreKindLabel($t, k.kind ?? "")}</option
                                >
                            {/each}
                        </Input>
                    </div>
                    <div class="mb-2">
                        <label for="keystore-name">{$t("keystores.name")}</label>
                        <Input id="keystore-name" type="text" required bind:value={newName} />
                    </div>
                    {#if kind?.references}
                        <p class="text-muted small">
                            {$t("keystores.references-help", { hint: kind.referenceHint ?? "" })}
                        </p>
                    {/if}
                    {#each kind?.form ?? [] as field (field.id)}
                        <ResourceInput
                            edit
                            index={"keystore-" + field.id}
                            specs={field as Field}
                            bind:value={newParams[field.id!]}
                        />
                    {/each}
                </form>
            </ModalBody>
            <ModalFooter>
                <Button
                    type="submit"
                    form="keystore-create-form"
                    color="primary"
                    disabled={creating || !newName}
                >
                    {#if creating}
                        <Spinner size="sm" />
                    {:else}
                        <i class="bi bi-plus-lg"></i>
                    {/if}
                    {$t("keystores.create")}
                </Button>
                <Button color="secondary" on:click={() => (createModalOpen = false)}>
                    {$t("common.cancel")}
                </Button>
            </ModalFooter>
        </Modal>
    {/if}
{/await}
