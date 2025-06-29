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

<script context="module" lang="ts">
    import type { ModalController } from "$lib/model/modal_controller";

    export const controls: ModalController = {
        Open() { },
    };
</script>

<script lang="ts">
    import { createEventDispatcher } from "svelte";

    import { Button, Modal, ModalBody, ModalFooter, ModalHeader, Spinner } from "@sveltestrap/sveltestrap";

    import { viewZone as APIViewZone } from "$lib/api/zone";
    import { t } from "$lib/translations";

    const dispatch = createEventDispatcher();

    export let isOpen = false;
    let inProgress = false;

    function Open(): void {
        isOpen = true;
        inProgress = false;
    }

    function toggle(): void {
        isOpen = !isOpen;
        inProgress = false;
    }

    function submit(): void {
        inProgress = true;
        dispatch("detachDomain");
    }

    controls.Open = Open;
</script>

<Modal {isOpen} size="lg" {toggle}>
    <ModalHeader class="bg-danger-subtle" {toggle}>{$t("domains.removal")}</ModalHeader>
    <ModalBody>
        {$t("domains.alert.remove")}
    </ModalBody>
    <ModalFooter>
        <Button
            outline
            color="secondary"
            disabled={inProgress}
            on:click={() => (isOpen = false)}
        >
            {$t("domains.view.cancel-title")}
        </Button>
        <Button
            color="danger"
            disabled={inProgress}
            on:click={submit}
        >
            {#if inProgress}
                <Spinner size="sm" />
            {/if}
            {$t("domains.discard")}
        </Button>
    </ModalFooter>
</Modal>
