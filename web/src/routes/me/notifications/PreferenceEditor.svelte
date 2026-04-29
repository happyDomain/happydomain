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
    import { untrack } from "svelte";

    import {
        Button,
        Input,
        Modal,
        ModalBody,
        ModalFooter,
        ModalHeader,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import {
        createPreference,
        updatePreference,
        type NotificationChannel,
        type NotificationPreference,
    } from "$lib/api/notifications";
    import type { HappydnsDomainWithCheckStatus, HappydnsStatus } from "$lib/api-base/types.gen";
    import { domains } from "$lib/stores/domains";
    import { toasts } from "$lib/stores/toasts";
    import { t } from "$lib/translations";
    import {
        StatusOK,
        StatusInfo,
        StatusWarn,
        StatusCrit,
        StatusError,
    } from "$lib/utils/checkers";

    interface Props {
        open: boolean;
        preference: NotificationPreference | null;
        channels: NotificationChannel[];
        onSave: () => void;
        onClose: () => void;
    }

    let { open, preference, channels, onSave, onClose }: Props = $props();

    type Scope = "global" | "domain" | "service";

    let scope: Scope = $state("global");
    let domainId: string = $state("");
    let serviceId: string = $state("");
    let channelIds: string[] = $state([]);
    let minStatus: HappydnsStatus = $state(StatusWarn);
    let notifyRecovery: boolean = $state(true);
    let enabled: boolean = $state(true);
    let quietHoursEnabled: boolean = $state(false);
    let quietStart: number = $state(22);
    let quietEnd: number = $state(7);
    let saving: boolean = $state(false);

    $effect(() => {
        if (!open) return;
        untrack(() => {
            const p = preference;
            if (p) {
                if (p.serviceId) {
                    scope = "service";
                    serviceId = p.serviceId;
                    domainId = p.domainId ?? "";
                } else if (p.domainId) {
                    scope = "domain";
                    domainId = p.domainId;
                    serviceId = "";
                } else {
                    scope = "global";
                    domainId = "";
                    serviceId = "";
                }
                channelIds = p.channelIds ? [...p.channelIds] : [];
                minStatus = p.minStatus ?? StatusWarn;
                notifyRecovery = p.notifyRecovery ?? true;
                enabled = p.enabled ?? true;
                if (
                    p.quietStart !== undefined &&
                    p.quietEnd !== undefined &&
                    p.quietStart !== null &&
                    p.quietEnd !== null
                ) {
                    quietHoursEnabled = true;
                    quietStart = p.quietStart;
                    quietEnd = p.quietEnd;
                } else {
                    quietHoursEnabled = false;
                    quietStart = 22;
                    quietEnd = 7;
                }
            } else {
                scope = "global";
                domainId = "";
                serviceId = "";
                channelIds = [];
                minStatus = StatusWarn;
                notifyRecovery = true;
                enabled = true;
                quietHoursEnabled = false;
                quietStart = 22;
                quietEnd = 7;
            }
        });
    });

    function toggleChannel(id: string) {
        channelIds = channelIds.includes(id)
            ? channelIds.filter((c) => c !== id)
            : [...channelIds, id];
    }

    async function save() {
        const payload: Parameters<typeof createPreference>[0] = {
            channelIds: channelIds.length ? channelIds : undefined,
            minStatus,
            notifyRecovery,
            enabled,
        };
        if (scope === "domain") {
            if (!domainId) return;
            payload.domainId = domainId;
        } else if (scope === "service") {
            if (!serviceId) return;
            payload.serviceId = serviceId;
            if (domainId) payload.domainId = domainId;
        }
        if (quietHoursEnabled) {
            payload.quietStart = quietStart;
            payload.quietEnd = quietEnd;
        }

        saving = true;
        try {
            if (preference?.id) {
                await updatePreference(preference.id, payload);
            } else {
                await createPreference(payload);
            }
            toasts.addToast({
                title: $t(
                    preference?.id
                        ? "settings.notifications.preferences.updated"
                        : "settings.notifications.preferences.created",
                ),
                timeout: 4000,
                type: "success",
            });
            onSave();
        } catch (e) {
            toasts.addErrorToast({
                title: $t("settings.notifications.preferences.saveError"),
                message: String(e),
                timeout: 8000,
            });
        } finally {
            saving = false;
        }
    }

    let domainList: HappydnsDomainWithCheckStatus[] = $derived($domains ?? []);
</script>

<Modal isOpen={open} toggle={onClose} size="lg">
    <ModalHeader toggle={onClose}>
        {preference?.id
            ? $t("settings.notifications.preferences.edit")
            : $t("settings.notifications.preferences.add")}
    </ModalHeader>
    <ModalBody>
        <form id="preference-editor-form" onsubmit={(e) => { e.preventDefault(); save(); }}>
            <fieldset class="mb-3">
                <legend class="form-label">
                    {$t("settings.notifications.preferences.scope.label")}
                </legend>
                <div class="form-check">
                    <input
                        class="form-check-input"
                        type="radio"
                        id="scope-global"
                        name="scope"
                        value="global"
                        bind:group={scope}
                    />
                    <label class="form-check-label" for="scope-global">
                        {$t("settings.notifications.preferences.scope.global")}
                    </label>
                </div>
                <div class="form-check">
                    <input
                        class="form-check-input"
                        type="radio"
                        id="scope-domain"
                        name="scope"
                        value="domain"
                        bind:group={scope}
                    />
                    <label class="form-check-label" for="scope-domain">
                        {$t("settings.notifications.preferences.scope.domain")}
                    </label>
                </div>
                <div class="form-check">
                    <input
                        class="form-check-input"
                        type="radio"
                        id="scope-service"
                        name="scope"
                        value="service"
                        bind:group={scope}
                    />
                    <label class="form-check-label" for="scope-service">
                        {$t("settings.notifications.preferences.scope.service")}
                    </label>
                </div>
            </fieldset>

            {#if scope === "domain" || scope === "service"}
                <div class="mb-3">
                    <label for="pref-domain" class="form-label">
                        {$t("settings.notifications.preferences.domain")}
                    </label>
                    <Input id="pref-domain" type="select" bind:value={domainId}>
                        <option value="">{$t("settings.notifications.preferences.selectDomain")}</option>
                        {#each domainList as d (d.id)}
                            <option value={d.id}>{d.domain}</option>
                        {/each}
                    </Input>
                </div>
            {/if}

            {#if scope === "service"}
                <div class="mb-3">
                    <label for="pref-service" class="form-label">
                        {$t("settings.notifications.preferences.serviceId")}
                    </label>
                    <Input id="pref-service" type="text" bind:value={serviceId} />
                    <small class="form-text text-muted">
                        {$t("settings.notifications.preferences.serviceIdHelp")}
                    </small>
                </div>
            {/if}

            <div class="mb-3">
                <label class="form-label" for="pref-channels">
                    {$t("settings.notifications.preferences.channels")}
                </label>
                {#if channels.length === 0}
                    <div class="alert alert-secondary mb-0 py-2">
                        {$t("settings.notifications.preferences.noChannels")}
                    </div>
                {:else}
                    <div id="pref-channels">
                        {#each channels as c (c.id)}
                            <div class="form-check">
                                <input
                                    class="form-check-input"
                                    type="checkbox"
                                    id={`pref-ch-${c.id}`}
                                    checked={channelIds.includes(c.id ?? "")}
                                    onchange={() => toggleChannel(c.id ?? "")}
                                />
                                <label class="form-check-label" for={`pref-ch-${c.id}`}>
                                    {c.name || c.type}
                                    <span class="text-muted">({c.type})</span>
                                </label>
                            </div>
                        {/each}
                    </div>
                    <small class="form-text text-muted">
                        {$t("settings.notifications.preferences.channelsHelp")}
                    </small>
                {/if}
            </div>

            <div class="mb-3">
                <label for="pref-min-status" class="form-label">
                    {$t("settings.notifications.preferences.minStatus")}
                </label>
                <Input id="pref-min-status" type="select" bind:value={minStatus}>
                    <option value={StatusOK}>{$t("checkers.status.ok")}</option>
                    <option value={StatusInfo}>{$t("checkers.status.info")}</option>
                    <option value={StatusWarn}>{$t("checkers.status.warning")}</option>
                    <option value={StatusCrit}>{$t("checkers.status.critical")}</option>
                    <option value={StatusError}>{$t("checkers.status.error")}</option>
                </Input>
                <small class="form-text text-muted">
                    {$t("settings.notifications.preferences.minStatusHelp")}
                </small>
            </div>

            <div class="form-check form-switch mb-3">
                <input
                    class="form-check-input"
                    type="checkbox"
                    role="switch"
                    id="pref-recovery"
                    bind:checked={notifyRecovery}
                />
                <label class="form-check-label" for="pref-recovery">
                    {$t("settings.notifications.preferences.notifyRecovery")}
                </label>
            </div>

            <div class="form-check form-switch mb-2">
                <input
                    class="form-check-input"
                    type="checkbox"
                    role="switch"
                    id="pref-quiet"
                    bind:checked={quietHoursEnabled}
                />
                <label class="form-check-label" for="pref-quiet">
                    {$t("settings.notifications.preferences.quietHours")}
                </label>
            </div>
            {#if quietHoursEnabled}
                <div class="d-flex gap-2 mb-3 align-items-end">
                    <div>
                        <label for="pref-quiet-start" class="form-label">
                            {$t("settings.notifications.preferences.quietStart")}
                        </label>
                        <Input
                            id="pref-quiet-start"
                            type="number"
                            min="0"
                            max="23"
                            bind:value={quietStart}
                        />
                    </div>
                    <div>
                        <label for="pref-quiet-end" class="form-label">
                            {$t("settings.notifications.preferences.quietEnd")}
                        </label>
                        <Input
                            id="pref-quiet-end"
                            type="number"
                            min="0"
                            max="23"
                            bind:value={quietEnd}
                        />
                    </div>
                    <small class="form-text text-muted">
                        {$t("settings.notifications.preferences.quietHoursHelp")}
                    </small>
                </div>
            {/if}

            <div class="form-check form-switch">
                <input
                    class="form-check-input"
                    type="checkbox"
                    role="switch"
                    id="pref-enabled"
                    bind:checked={enabled}
                />
                <label class="form-check-label" for="pref-enabled">
                    {$t("settings.notifications.preferences.enabled")}
                </label>
            </div>
        </form>
    </ModalBody>
    <ModalFooter>
        <Button type="button" color="secondary" outline on:click={onClose} disabled={saving}>
            {$t("common.cancel")}
        </Button>
        <Button
            type="submit"
            form="preference-editor-form"
            color="primary"
            disabled={saving ||
                (scope === "domain" && !domainId) ||
                (scope === "service" && !serviceId)}
        >
            {#if saving}<Spinner size="sm" class="me-2" />{/if}
            {$t("common.save")}
        </Button>
    </ModalFooter>
</Modal>
