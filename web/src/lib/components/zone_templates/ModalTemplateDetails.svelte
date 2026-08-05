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

<script module lang="ts">
    import type { ModalController } from "$lib/model/modal_controller";

    export const controls: ModalController = {
        Open() {},
    };
</script>

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

    import type { ZoneTemplate, ZoneTemplateInput } from "$lib/model/zone_template";
    import { t } from "$lib/translations";
    import TemplateFields from "./TemplateFields.svelte";

    interface Props {
        template: ZoneTemplate;
        // Saves the details changed, giving the template saved.
        save: (input: ZoneTemplateInput) => Promise<ZoneTemplate>;
        isOpen?: boolean;
    }

    let { template, save, isOpen = $bindable(false) }: Props = $props();

    let name = $state("");
    let description = $state("");
    let inProgress = $state(false);
    let error = $state<string | null>(null);

    async function submit(e: SubmitEvent) {
        e.preventDefault();
        if (inProgress || !name.trim()) return;

        inProgress = true;
        error = null;
        try {
            await save({ name: name.trim(), description: description.trim() });
            isOpen = false;
        } catch (err) {
            error = err instanceof Error ? err.message : String(err);
        } finally {
            inProgress = false;
        }
    }

    function Open(): void {
        name = template.name;
        description = template.description ?? "";
        error = null;
        inProgress = false;
        isOpen = true;
    }

    function toggle(): void {
        isOpen = !isOpen;
    }

    controls.Open = Open;
</script>

<Modal {isOpen} {toggle}>
    <ModalHeader {toggle}>{$t("zone-templates.edit")}</ModalHeader>
    <ModalBody>
        <form id="templateDetailsForm" onsubmit={submit}>
            <TemplateFields
                id="template-details"
                bind:name
                bind:description
                disabled={inProgress}
            />
            {#if error}
                <Alert color="danger" fade={false} class="mb-0">{error}</Alert>
            {/if}
        </form>
    </ModalBody>
    <ModalFooter>
        <Button outline color="secondary" on:click={() => (isOpen = false)}>
            {$t("common.cancel")}
        </Button>
        <Button
            color="success"
            type="submit"
            form="templateDetailsForm"
            disabled={inProgress || !name.trim()}
        >
            {#if inProgress}
                <Spinner size="sm" />
            {/if}
            {$t("zone-templates.save")}
        </Button>
    </ModalFooter>
</Modal>
