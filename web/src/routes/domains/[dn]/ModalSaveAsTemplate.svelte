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
    import { goto } from "$app/navigation";
    import { templateLinks } from "$links";

    import {
        Alert,
        Button,
        Modal,
        ModalBody,
        ModalFooter,
        ModalHeader,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { createZoneTemplate } from "$lib/api/zone_template";
    import TemplateFields from "$lib/components/zone_templates/TemplateFields.svelte";
    import type { Domain } from "$lib/model/domain";
    import { toasts } from "$lib/stores/toasts";
    import { upsertZoneTemplate } from "$lib/stores/zone_templates";
    import { t } from "$lib/translations";

    interface Props {
        domain: Domain;
        // The zone shown, saved as a template; the one being edited when none.
        selectedHistory?: string;
        isOpen?: boolean;
    }

    let { domain, selectedHistory, isOpen = $bindable(false) }: Props = $props();

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
            const tpl = await createZoneTemplate({
                name: name.trim(),
                description: description.trim(),
                domain_id: domain.id,
                zone_id: selectedHistory || undefined,
            });
            upsertZoneTemplate(tpl);
            isOpen = false;
            toasts.addToast({
                type: "success",
                title: $t("zone-templates.save-as.done", { name: tpl.name }),
                message: $t("zone-templates.save-as.open"),
                timeout: 8000,
                onclick: () => goto(templateLinks().template(encodeURIComponent(tpl.id))),
            });
        } catch (err) {
            error = err instanceof Error ? err.message : String(err);
        } finally {
            inProgress = false;
        }
    }

    function Open(): void {
        name = "";
        description = "";
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
    <ModalHeader {toggle}>{$t("zone-templates.save-as.title")}</ModalHeader>
    <ModalBody>
        <p class="text-muted">
            {$t("zone-templates.save-as.description", { domain: domain.domain })}
        </p>
        <form id="saveAsTemplateForm" onsubmit={submit}>
            <TemplateFields
                id="save-as-template"
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
            color="primary"
            type="submit"
            form="saveAsTemplateForm"
            disabled={inProgress || !name.trim()}
        >
            {#if inProgress}
                <Spinner size="sm" />
            {/if}
            {$t("zone-templates.save-as.submit")}
        </Button>
    </ModalFooter>
</Modal>
