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

<script module lang="ts">
    import type { ModalController } from "$lib/model/modal_controller";

    export const controls: ModalController = {
        Open() { },
    };
</script>

<script lang="ts">
    import { createEventDispatcher } from "svelte";

    import {
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

    import { importZone as APIImportZone } from "$lib/api/zone";
    import type { Domain } from "$lib/model/domain";
    import { t } from "$lib/translations";

    const dispatch = createEventDispatcher();

    interface Props {
        domain: Domain;
        selectedHistory?: string;
        isOpen?: boolean;
    }

    let { domain, selectedHistory = "", isOpen = $bindable(false) }: Props = $props();

    let uploadInProgress = $state(false);
    let zoneImportContent = $state("");
    let zoneImportFiles: FileList | undefined = $state();
    let uploadModalActiveTab: string | number = $state(0);

    function importZone(): void {
        uploadInProgress = true;
        let file = new Blob([zoneImportContent], { type: "text/plain" });
        if (uploadModalActiveTab != "uploadText" && zoneImportFiles?.[0]) {
            file = zoneImportFiles[0];
        }
        APIImportZone(domain, selectedHistory, file).then(
            (v) => {
                isOpen = false;
                dispatch("retrieveZoneDone", v);
            },
            (err: any) => {
                uploadInProgress = false;
                throw err;
            },
        );
    }

    function Open(): void {
        isOpen = true;
        zoneImportContent = "";
        uploadModalActiveTab = 0;
    }

    function toggle(): void {
        isOpen = !isOpen;
    }

    controls.Open = Open;
</script>

<Modal {isOpen} size="lg" {toggle}>
    <ModalHeader class="bg-info-subtle" {toggle}>{$t("zones.upload")}</ModalHeader>
    <ModalBody>
        <TabContent on:tab={(e) => (uploadModalActiveTab = e.detail)}>
            <TabPane tabId="uploadText" tab={$t("zones.import-text")} active>
                <Input
                    class="mt-3"
                    type="textarea"
                    style="height: 200px;"
                    placeholder="@         4269 IN SOA   root ns 2042070136 ..."
                    bind:value={zoneImportContent}
                />
            </TabPane>
            <TabPane tabId="uploadFile" tab={$t("zones.import-file")}>
                {#if isOpen}
                    <Input class="mt-3" type="file" bind:files={zoneImportFiles} />
                {/if}
            </TabPane>
        </TabContent>
    </ModalBody>
    <ModalFooter>
        <Button outline color="secondary" on:click={() => (isOpen = false)}>
            {$t("common.cancel")}
        </Button>
        <Button color="primary" disabled={uploadInProgress} on:click={importZone}>
            {#if uploadInProgress}
                <Spinner size="sm" />
            {/if}
            {$t("domains.actions.upload")}
        </Button>
    </ModalFooter>
</Modal>
