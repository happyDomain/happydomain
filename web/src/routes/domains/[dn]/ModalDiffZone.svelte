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

    import {
        Button,
        Icon,
        Input,
        Modal,
        ModalBody,
        ModalFooter,
        ModalHeader,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { applyZone as APIApplyZone } from "$lib/api/zone";
    import type { Domain } from "$lib/model/domain";
    import DiffZone from "$lib/components/zones/DiffZone.svelte";
    import { t } from "$lib/translations";

    const dispatch = createEventDispatcher();

    export let domain: Domain;
    export let selectedHistory: string = "";
    export let isOpen = false;

    let zoneDiffLength = 0;
    let zoneDiffCreated = 0;
    let zoneDiffDeleted = 0;
    let zoneDiffModified = 0;

    let selectedDiff: Array<string> | null = null;
    let diffCommitMsg = "";
    let selectedDiffCreated = 0;
    let selectedDiffDeleted = 0;
    let selectedDiffModified = 0;

    function Open(): void {
        zoneDiffLength = 0;
        selectedDiff = null;
        isOpen = true;
        propagationInProgress = false;
        diffCommitMsg = "";
    }

    function receiveError(evt: CustomEvent): void {
        isOpen = false;
        throw evt.detail;
    }

    function computedDiff(evt: CustomEvent): void {
        zoneDiffLength = evt.detail.zoneDiffLength;
        zoneDiffCreated = evt.detail.zoneDiffCreated;
        zoneDiffDeleted = evt.detail.zoneDiffDeleted;
        zoneDiffModified = evt.detail.zoneDiffModified;
    }

    function computedSelection(evt: CustomEvent): void {
        selectedDiffCreated = evt.detail.selectedDiffCreated;
        selectedDiffDeleted = evt.detail.selectedDiffDeleted;
        selectedDiffModified = evt.detail.selectedDiffModified;
    }

    let propagationInProgress = false;
    async function applyDiff() {
        if (!domain || !selectedHistory || !selectedDiff) return;

        propagationInProgress = true;
        try {
            dispatch(
                "retrieveZoneDone",
                await APIApplyZone(domain, selectedHistory, selectedDiff, diffCommitMsg),
            );
        } finally {
            isOpen = false;
        }
    }

    function toggle(): void {
        isOpen = !isOpen;
    }

    controls.Open = Open;
</script>

<Modal {isOpen} size="lg" scrollable {toggle}>
    {#if domain}
        <ModalHeader {toggle} class="bg-warning-subtle">
            {@html $t("domains.view.description", {
                domain: `<span class="font-monospace">${escape(domain.domain)}</span>`,
            })}
        </ModalHeader>
    {/if}
    <ModalBody>
        <DiffZone
            {domain}
            selectable
            bind:selectedDiff
            zoneFrom={selectedHistory}
            zoneTo="@"
            on:computed-diff={computedDiff}
            on:computed-selection={computedSelection}
            on:error={receiveError}
        >
            <div slot="nodiff" class="d-flex gap-3 align-items-center justify-content-center">
                <Icon name="check2-all" class="display-5 text-success" />
                {$t("domains.apply.nochange")}
            </div>
        </DiffZone>
    </ModalBody>
    <ModalFooter>
        {#if zoneDiffLength > 0}
            <Input
                id="commitmsg"
                placeholder={$t("domains.commit-msg")}
                bsSize="sm"
                bind:value={diffCommitMsg}
            />
            {#if zoneDiffCreated}
                <span class="text-success">
                    {$t("domains.apply.additions", { count: selectedDiffCreated })}
                </span>
            {/if}
            {#if zoneDiffCreated && zoneDiffDeleted}
                &ndash;
            {/if}
            {#if zoneDiffDeleted}
                <span class="text-danger">
                    {$t("domains.apply.deletions", { count: selectedDiffDeleted })}
                </span>
            {/if}
            {#if (zoneDiffCreated || zoneDiffDeleted) && zoneDiffModified}
                &ndash;
            {/if}
            {#if zoneDiffModified}
                <span class="text-warning">
                    {$t("domains.apply.modifications", { count: selectedDiffModified })}
                </span>
            {/if}
            {#if (zoneDiffCreated || zoneDiffDeleted || zoneDiffModified) && zoneDiffLength - zoneDiffCreated - zoneDiffDeleted - zoneDiffModified !== 0}
                &ndash;
            {/if}
            {#if selectedDiff && zoneDiffLength - zoneDiffCreated - zoneDiffDeleted - zoneDiffModified !== 0}
                <span class="text-info">
                    {$t("domains.apply.others", {
                        count:
                            selectedDiff.length -
                            selectedDiffCreated -
                            selectedDiffDeleted -
                            selectedDiffModified,
                    })}
                </span>
            {/if}
        {/if}
        <div class="d-flex gap-1">
            <Button outline color="secondary" on:click={() => (isOpen = false)}>
                {$t("common.cancel")}
            </Button>
            <Button
                color="success"
                disabled={propagationInProgress ||
                    !zoneDiffLength ||
                    !selectedDiff ||
                    selectedDiff.length === 0}
                on:click={applyDiff}
            >
                {#if propagationInProgress}
                    <Spinner size="sm" />
                {/if}
                {$t("domains.apply.button")}
            </Button>
        </div>
    </ModalFooter>
</Modal>
