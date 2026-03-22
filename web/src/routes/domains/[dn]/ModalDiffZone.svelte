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
        Open() {},
    };
</script>

<script lang="ts">
    import {
        Alert,
        Button,
        Icon,
        Input,
        Modal,
        ModalBody,
        ModalFooter,
        ModalHeader,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { applyZone as APIApplyZone, prepareZone as APIPrepareZone } from "$lib/api/zone";
    import type { FullCorrection } from "$lib/model/correction";
    import type { Domain } from "$lib/model/domain";
    import {
        ApplyConfirmUnexpected,
        ApplyConfirmAlways,
        ApplyConfirmNever,
    } from "$lib/model/usersettings";
    import DiffZone from "$lib/components/zones/DiffZone.svelte";
    import DiffZoneView from "$lib/components/zones/DiffZoneView.svelte";
    import { invalidateZoneDiff } from "$lib/stores/zonediff";
    import { userSession } from "$lib/stores/usersession";
    import { getZone, thisZone } from "$lib/stores/thiszone";
    import { t } from "$lib/translations";

    interface Props {
        domain: Domain;
        selectedHistory?: string;
        isOpen?: boolean;
    }

    let { domain, selectedHistory = "", isOpen = $bindable(false) }: Props = $props();

    let zoneDiffLength = $state(0);
    let zoneDiffCreated = $state(0);
    let zoneDiffDeleted = $state(0);
    let zoneDiffModified = $state(0);

    let selectedDiff: Array<string> | null = $state(null);
    let diffCommitMsg = $state("");
    let selectedDiffCreated = $state(0);
    let selectedDiffDeleted = $state(0);
    let selectedDiffModified = $state(0);

    let preparePhase: "select" | "confirm" = $state("select");
    let prepareResponse: {
        corrections: Array<FullCorrection>;
        nbDiffs: number;
    } | null = $state(null);
    let prepareInProgress = $state(false);

    function Open(): void {
        zoneDiffLength = 0;
        selectedDiff = null;
        isOpen = true;
        propagationInProgress = false;
        preparePhase = "select";
        prepareResponse = null;
        prepareInProgress = false;
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

    let propagationInProgress = $state(false);

    async function doApply() {
        if (!domain || !selectedHistory || !selectedDiff) return;

        propagationInProgress = true;
        try {
            await APIApplyZone(domain, selectedHistory, selectedDiff, diffCommitMsg);
            invalidateZoneDiff();
            if ($thisZone)
                getZone(domain, $thisZone.id).then(() => {
                    invalidateZoneDiff();
                });
        } finally {
            isOpen = false;
        }
    }

    async function applyDiff() {
        if (!domain || !selectedHistory || !selectedDiff) return;

        const setting = $userSession?.settings?.applyconfirm ?? ApplyConfirmUnexpected;

        if (setting === ApplyConfirmNever) {
            return doApply();
        }

        prepareInProgress = true;
        try {
            const resp = await APIPrepareZone(domain, selectedHistory, selectedDiff);
            prepareResponse = resp;

            if (setting === ApplyConfirmAlways) {
                preparePhase = "confirm";
            } else {
                // UNEXPECTED: show confirmation only if counts differ
                if (resp.nbDiffs !== selectedDiff.length) {
                    preparePhase = "confirm";
                } else {
                    return doApply();
                }
            }
        } finally {
            prepareInProgress = false;
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
        {#if propagationInProgress}
            <div class="my-2 text-center">
                <Spinner color="success" />
                <p>{$t("wait.propagation")}</p>
            </div>
        {:else if prepareInProgress}
            <div class="my-2 text-center">
                <Spinner color="warning" />
                <p>{$t("wait.preparation")}</p>
            </div>
        {:else if preparePhase === "select"}
            <DiffZone
                {domain}
                selectable
                disabled={prepareInProgress || propagationInProgress}
                bind:selectedDiff
                zoneFrom={selectedHistory}
                zoneTo="@"
                on:computed-diff={computedDiff}
                on:computed-selection={computedSelection}
                on:error={receiveError}
            >
                {#snippet nodiff()}
                    <div class="d-flex gap-3 align-items-center justify-content-center">
                        <Icon name="check2-all" class="display-5 text-success" />
                        {$t("domains.apply.nochange")}
                    </div>
                {/snippet}
            </DiffZone>
        {:else if preparePhase === "confirm" && prepareResponse}
            {#if selectedDiff && prepareResponse.nbDiffs !== selectedDiff.length}
                <Alert color="warning">
                    <Icon name="exclamation-triangle-fill" class="me-2" />
                    {$t("domains.apply.prepare-warning")}
                </Alert>
            {/if}
            <p>
                {$t("domains.apply.prepare-info", {
                    nbDiffs: prepareResponse.nbDiffs,
                    nbSelected: selectedDiff?.length ?? 0,
                })}
            </p>
            <DiffZoneView
                disabled={prepareInProgress || propagationInProgress}
                zoneDiff={prepareResponse.corrections}
            >
                {#snippet nodiff()}
                    <div class="d-flex gap-2 align-items-center">
                        <Icon name="exclamation" class="display-5 text-warning" />
                        {$t("domains.apply.change-already-applied")}
                    </div>
                {/snippet}
            </DiffZoneView>
            {#if selectedDiff && prepareResponse.nbDiffs === selectedDiff.length}
                <hr />
                <p class="text-muted small mb-0">
                    {$t("domains.apply.double-check")}
                </p>
            {/if}
        {/if}
    </ModalBody>
    <ModalFooter>
        {#if preparePhase === "select"}
            {#if zoneDiffLength > 0 && !(prepareInProgress || propagationInProgress)}
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
                        prepareInProgress ||
                        !zoneDiffLength ||
                        !selectedDiff ||
                        selectedDiff.length === 0}
                    on:click={applyDiff}
                >
                    {#if propagationInProgress || prepareInProgress}
                        <Spinner size="sm" />
                    {/if}
                    {($userSession?.settings?.applyconfirm ?? ApplyConfirmUnexpected) ===
                    ApplyConfirmAlways
                        ? $t("common.next")
                        : $t("domains.apply.button")}
                </Button>
            </div>
        {:else if preparePhase === "confirm"}
            <div class="d-flex gap-1">
                <Button outline color="secondary" on:click={() => (preparePhase = "select")}>
                    {$t("domains.apply.back")}
                </Button>
                <Button
                    color="success"
                    disabled={propagationInProgress ||
                        !prepareResponse ||
                        prepareResponse.nbDiffs == 0}
                    on:click={doApply}
                >
                    {#if propagationInProgress}
                        <Spinner size="sm" />
                    {/if}
                    {$t("domains.apply.confirm")}
                </Button>
            </div>
        {/if}
    </ModalFooter>
</Modal>
