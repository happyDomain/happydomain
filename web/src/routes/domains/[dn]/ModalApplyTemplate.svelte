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
    import { resolve } from "$app/paths";

    import {
        Alert,
        Button,
        FormGroup,
        Input,
        Label,
        ModalBody,
        ModalFooter,
        ModalHeader,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import Modal from "$lib/components/Modal.svelte";

    import { applyZoneTemplate } from "$lib/api/zone_template";
    import type { Domain } from "$lib/model/domain";
    import type { Zone } from "$lib/model/zone";
    import {
        countServices,
        isGlobalTemplate,
        type ZoneTemplateApplyMode,
    } from "$lib/model/zone_template";
    import { thisZone } from "$lib/stores/thiszone";
    import { toasts } from "$lib/stores/toasts";
    import { refreshZoneTemplates, zoneTemplates } from "$lib/stores/zone_templates";
    import { t } from "$lib/translations";
    import { controls as ctrlDiffZone } from "./ModalDiffZone.svelte";

    interface Props {
        domain: Domain;
        // The zone shown; the template goes to the one being edited.
        selectedHistory?: string;
        // Tells the page the zone changed: a new one when it had to be
        // derived from a published one.
        onapplied: (zone: Zone) => void;
        isOpen?: boolean;
    }

    let { domain, selectedHistory, onapplied, isOpen = $bindable(false) }: Props = $props();

    let templateId = $state("");
    let mode = $state<ZoneTemplateApplyMode>("merge");
    let loading = $state(false);
    let inProgress = $state(false);
    let error = $state<string | null>(null);

    let mine = $derived($zoneTemplates?.filter((tpl) => !isGlobalTemplate(tpl)) ?? []);
    let globals = $derived($zoneTemplates?.filter((tpl) => isGlobalTemplate(tpl)) ?? []);
    let chosen = $derived($zoneTemplates?.find((tpl) => tpl.id === templateId));

    async function submit(e: SubmitEvent) {
        e.preventDefault();
        const editing = domain.zone_history?.[0];
        if (inProgress || !chosen || !editing) return;

        inProgress = true;
        error = null;
        try {
            const zone = await applyZoneTemplate(domain, editing, { template_id: chosen.id, mode });
            if (zone.id === selectedHistory) thisZone.set(zone);
            isOpen = false;
            toasts.addToast({
                type: "success",
                title: $t("zone-templates.apply.done", { name: chosen.name }),
                message: $t("zone-templates.apply.review"),
                timeout: 10000,
                onclick: () => ctrlDiffZone.Open(),
            });
            onapplied(zone);
        } catch (err) {
            error = err instanceof Error ? err.message : String(err);
        } finally {
            inProgress = false;
        }
    }

    function Open(): void {
        templateId = "";
        mode = "merge";
        error = null;
        inProgress = false;
        isOpen = true;

        // Templates may have been added from another page.
        loading = true;
        refreshZoneTemplates()
            .then((list) => {
                if (!templateId && list.length > 0) templateId = list[0].id;
            })
            .catch((err) => (error = err instanceof Error ? err.message : String(err)))
            .finally(() => (loading = false));
    }

    function toggle(): void {
        isOpen = !isOpen;
    }

    controls.Open = Open;
</script>

<Modal {isOpen} {toggle}>
    <ModalHeader {toggle}>{$t("zone-templates.apply.title")}</ModalHeader>
    <ModalBody>
        <p class="text-muted">
            {$t("zone-templates.apply.description", { domain: domain.domain })}
        </p>
        {#if loading && $zoneTemplates === undefined}
            <div class="text-center"><Spinner size="sm" /></div>
        {:else if $zoneTemplates?.length === 0}
            <p class="mb-0">
                {$t("zone-templates.apply.none")}
                <a href={resolve("/templates")}>{$t("zone-templates.apply.manage")}</a>
            </p>
        {:else}
            <form id="applyTemplateForm" onsubmit={submit}>
                <FormGroup>
                    <Label for="apply-template-choice">{$t("zone-templates.apply.template")}</Label>
                    <Input
                        id="apply-template-choice"
                        type="select"
                        bind:value={templateId}
                        disabled={inProgress}
                    >
                        {#if mine.length > 0}
                            <optgroup label={$t("zone-templates.mine")}>
                                {#each mine as tpl (tpl.id)}
                                    <option value={tpl.id}>{tpl.name}</option>
                                {/each}
                            </optgroup>
                        {/if}
                        {#if globals.length > 0}
                            <optgroup label={$t("zone-templates.global")}>
                                {#each globals as tpl (tpl.id)}
                                    <option value={tpl.id}>{tpl.name}</option>
                                {/each}
                            </optgroup>
                        {/if}
                    </Input>
                    {#if chosen}
                        <div class="form-text">
                            {#if chosen.description}{chosen.description} ·{/if}
                            {$t("zone-templates.services", { count: countServices(chosen) })}
                        </div>
                    {/if}
                </FormGroup>
                <fieldset>
                    <legend class="form-label fs-6">{$t("zone-templates.apply.mode")}</legend>
                    {#each ["merge", "overwrite"] as const as m (m)}
                        <div class="form-check mb-2">
                            <input
                                class="form-check-input"
                                type="radio"
                                name="apply-template-mode"
                                id="apply-template-{m}"
                                value={m}
                                bind:group={mode}
                                disabled={inProgress}
                            />
                            <label class="form-check-label" for="apply-template-{m}">
                                <strong>{$t(`zone-templates.apply.${m}`)}</strong>
                                <span class="d-block small text-muted">
                                    {$t(`zone-templates.apply.${m}-help`)}
                                </span>
                            </label>
                        </div>
                    {/each}
                </fieldset>
            </form>
        {/if}
        {#if error}
            <Alert color="danger" fade={false} class="mt-3 mb-0">{error}</Alert>
        {/if}
    </ModalBody>
    <ModalFooter>
        <Button outline color="secondary" on:click={() => (isOpen = false)}>
            {$t("common.cancel")}
        </Button>
        <Button
            color={mode === "overwrite" ? "danger" : "primary"}
            type="submit"
            form="applyTemplateForm"
            disabled={inProgress || !chosen}
        >
            {#if inProgress}
                <Spinner size="sm" />
            {/if}
            {$t("zone-templates.apply.submit")}
        </Button>
    </ModalFooter>
</Modal>
