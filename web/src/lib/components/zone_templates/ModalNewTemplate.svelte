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
    export const controls = {
        // Open starts from nothing; OpenImport from a zone file.
        Open(): void {},
        OpenImport(): void {},
    };
</script>

<script lang="ts">
    import { goto } from "$app/navigation";
    import { templateLinks } from "$links";

    import {
        Alert,
        Button,
        Input,
        Modal,
        ModalBody,
        ModalFooter,
        ModalHeader,
        Spinner,
        TabContent,
        TabPane,
    } from "@sveltestrap/sveltestrap";

    import type { ZoneTemplate, ZoneTemplateImportInput } from "$lib/model/zone_template";
    import { t } from "$lib/translations";
    import TemplateFields from "./TemplateFields.svelte";

    interface Props {
        // Makes the template: empty, or from the zone file given, written
        // for the origin of the input.
        create: (input: ZoneTemplateImportInput, zoneFile?: Blob) => Promise<ZoneTemplate>;
        isOpen?: boolean;
    }

    let { create, isOpen = $bindable(false) }: Props = $props();

    // Importing a zone file, rather than starting from nothing.
    let importing = $state(false);
    let name = $state("");
    let description = $state("");
    let origin = $state("");
    let zoneText = $state("");
    let zoneFiles: FileList | undefined = $state();
    let activeTab: string | number = $state("text");

    let inProgress = $state(false);
    let error = $state<string | null>(null);

    function zoneFile(): Blob {
        if (activeTab === "file" && zoneFiles?.[0]) return zoneFiles[0];
        return new Blob([zoneText], { type: "text/plain" });
    }

    async function submit(e: SubmitEvent) {
        e.preventDefault();
        if (inProgress || !name.trim()) return;

        inProgress = true;
        error = null;
        try {
            const input = { name: name.trim(), description: description.trim() };
            const tpl = await (importing
                ? create({ ...input, origin: origin.trim() }, zoneFile())
                : create(input));
            isOpen = false;
            goto(templateLinks().template(encodeURIComponent(tpl.id)));
        } catch (err) {
            error = err instanceof Error ? err.message : String(err);
        } finally {
            inProgress = false;
        }
    }

    function open(withZoneFile: boolean): void {
        importing = withZoneFile;
        name = "";
        description = "";
        origin = "";
        zoneText = "";
        zoneFiles = undefined;
        activeTab = "text";
        error = null;
        inProgress = false;
        isOpen = true;
    }

    controls.Open = () => open(false);
    controls.OpenImport = () => open(true);

    function toggle(): void {
        isOpen = !isOpen;
    }
</script>

<Modal {isOpen} size={importing ? "lg" : undefined} {toggle}>
    <ModalHeader {toggle}>
        {importing ? $t("zone-templates.import") : $t("zone-templates.new")}
    </ModalHeader>
    <ModalBody>
        <form id="newTemplateForm" onsubmit={submit}>
            <TemplateFields
                id="new-template"
                bind:name
                bind:description
                bind:origin
                withOrigin={importing}
                disabled={inProgress}
            />
            {#if importing}
                <p class="mb-1 fw-semibold">{$t("zone-templates.fields.zone")}</p>
                <TabContent on:tab={(e) => (activeTab = e.detail)}>
                    <TabPane tabId="text" tab={$t("zones.import-text")} active>
                        <Input
                            class="mt-3 font-monospace"
                            type="textarea"
                            style="height: 200px;"
                            aria-label={$t("zones.import-text")}
                            placeholder="www  3600 IN A  192.0.2.1"
                            bind:value={zoneText}
                            disabled={inProgress}
                        />
                    </TabPane>
                    <TabPane tabId="file" tab={$t("zones.import-file")}>
                        {#if isOpen}
                            <Input class="mt-3" type="file" bind:files={zoneFiles} />
                        {/if}
                    </TabPane>
                </TabContent>
                <p class="form-text">{$t("zone-templates.fields.zone-help")}</p>
            {/if}
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
            form="newTemplateForm"
            disabled={inProgress || !name.trim()}
        >
            {#if inProgress}
                <Spinner size="sm" />
            {/if}
            {importing ? $t("zone-templates.import-submit") : $t("zone-templates.create")}
        </Button>
    </ModalFooter>
</Modal>
