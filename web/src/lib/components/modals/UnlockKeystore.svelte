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
        Alert,
        Button,
        ModalBody,
        ModalFooter,
        ModalHeader,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import Modal from "$lib/components/Modal.svelte";
    import { unlockKeystore } from "$lib/api/keystores";
    import ResourceInput from "$lib/components/inputs/basic.svelte";
    import type { Field } from "$lib/model/custom_form.svelte";
    import { refreshKeystores } from "$lib/stores/keystores";
    import { settleUnlock, unlockRequest } from "$lib/stores/unlock";
    import { t } from "$lib/translations";

    let material: Record<string, unknown> = $state({});
    let error: string | null = $state(null);
    let unlocking = $state(false);

    // Reset per request: what was typed for one keystore is never sent to the next.
    $effect(() => {
        if ($unlockRequest) {
            material = {};
            error = null;
        }
    });

    async function unlock() {
        const ref = $unlockRequest?.locked.ref;
        if (!ref) return;
        unlocking = true;
        error = null;
        try {
            await unlockKeystore(ref, material);
            refreshKeystores().catch(() => {});
            settleUnlock(true);
        } catch (e) {
            error = e instanceof Error ? e.message : String(e);
        } finally {
            unlocking = false;
        }
    }
</script>

<Modal isOpen={$unlockRequest !== null} toggle={() => settleUnlock(false)}>
    <ModalHeader toggle={() => settleUnlock(false)}>
        <i class="bi bi-safe"></i>
        {$t("keystores.unlock-title", {
            name: $unlockRequest?.locked.name || $unlockRequest?.locked.kind || "",
        })}
    </ModalHeader>
    <form
        onsubmit={(e) => {
            e.preventDefault();
            unlock();
        }}
    >
        <ModalBody>
            <p>{$t("keystores.unlock-description")}</p>
            {#each $unlockRequest?.locked.form ?? [] as field (field.id)}
                <ResourceInput
                    edit
                    index={"unlock-" + field.id}
                    specs={field as Field}
                    bind:value={material[field.id!]}
                />
            {/each}
            {#if error}
                <Alert color="danger" class="mb-0">{error}</Alert>
            {/if}
        </ModalBody>
        <ModalFooter>
            <Button type="button" color="secondary" outline on:click={() => settleUnlock(false)}>
                {$t("common.cancel")}
            </Button>
            <Button type="submit" color="primary" disabled={unlocking}>
                {#if unlocking}
                    <Spinner size="sm" />
                {:else}
                    <i class="bi bi-unlock"></i>
                {/if}
                {$t("keystores.unlock")}
            </Button>
        </ModalFooter>
    </form>
</Modal>
