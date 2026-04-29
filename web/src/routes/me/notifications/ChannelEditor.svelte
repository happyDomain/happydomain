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

    import { createChannel, updateChannel, type NotificationChannel } from "$lib/api/notifications";
    import { toasts } from "$lib/stores/toasts";
    import { t } from "$lib/translations";

    import {
        getChannelConfigSchema,
        emptyConfigForSchema,
        type ChannelConfigField,
    } from "./channelConfigs";

    interface Props {
        open: boolean;
        channelTypes: string[];
        channel: NotificationChannel | null;
        onSave: () => void;
        onClose: () => void;
    }

    let { open, channelTypes, channel, onSave, onClose }: Props = $props();

    let type: string = $state("");
    let name: string = $state("");
    let enabled: boolean = $state(true);
    let config: Record<string, unknown> = $state({});
    let rawJson: string = $state("{}");
    let rawJsonError: string | null = $state(null);
    let saving: boolean = $state(false);
    let revealedSecrets: Record<string, boolean> = $state({});

    // Reset only on open transition; other props read via untrack so type loading doesn't retrigger.
    $effect(() => {
        if (!open) return;
        untrack(() => {
            const c = channel;
            const types = channelTypes;
            if (c) {
                type = c.type ?? "";
                name = c.name ?? "";
                enabled = c.enabled ?? true;
                config = (c.config ?? {}) as Record<string, unknown>;
            } else {
                type = types[0] ?? "";
                name = "";
                enabled = true;
                const schema = getChannelConfigSchema(type);
                config = schema ? emptyConfigForSchema(schema) : {};
            }
            rawJson = JSON.stringify(config, null, 2);
            rawJsonError = null;
            revealedSecrets = {};
        });
    });

    function onTypeChange() {
        // Reset to schema defaults so we never send fields that don't apply.
        const schema = getChannelConfigSchema(type);
        config = schema ? emptyConfigForSchema(schema) : {};
        rawJson = JSON.stringify(config, null, 2);
        rawJsonError = null;
    }

    let schema = $derived(getChannelConfigSchema(type));

    function setHeaderKey(field: ChannelConfigField, oldKey: string, newKey: string) {
        const cur = (config[field.key] as Record<string, string>) ?? {};
        const next: Record<string, string> = {};
        for (const [k, v] of Object.entries(cur)) {
            next[k === oldKey ? newKey : k] = v;
        }
        config = { ...config, [field.key]: next };
    }

    function setHeaderValue(field: ChannelConfigField, key: string, value: string) {
        const cur = (config[field.key] as Record<string, string>) ?? {};
        config = { ...config, [field.key]: { ...cur, [key]: value } };
    }

    function addHeader(field: ChannelConfigField) {
        const cur = (config[field.key] as Record<string, string>) ?? {};
        if ("" in cur) return;
        config = { ...config, [field.key]: { ...cur, "": "" } };
    }

    function removeHeader(field: ChannelConfigField, key: string) {
        const cur = { ...((config[field.key] as Record<string, string>) ?? {}) };
        delete cur[key];
        config = { ...config, [field.key]: cur };
    }

    async function save() {
        let payloadConfig: Record<string, unknown>;

        if (schema) {
            payloadConfig = config;
        } else {
            try {
                payloadConfig = JSON.parse(rawJson);
                rawJsonError = null;
            } catch (e) {
                rawJsonError = (e as Error).message;
                return;
            }
        }

        saving = true;
        try {
            if (channel?.id) {
                await updateChannel(channel.id, {
                    type,
                    name,
                    enabled,
                    config: payloadConfig,
                });
            } else {
                await createChannel({
                    type,
                    name,
                    enabled,
                    config: payloadConfig,
                });
            }
            toasts.addToast({
                title: $t(
                    channel?.id
                        ? "settings.notifications.channels.updated"
                        : "settings.notifications.channels.created",
                ),
                timeout: 4000,
                type: "success",
            });
            onSave();
        } catch (e) {
            toasts.addErrorToast({
                title: $t("settings.notifications.channels.saveError"),
                message: String(e),
                timeout: 8000,
            });
        } finally {
            saving = false;
        }
    }
</script>

<Modal isOpen={open} toggle={onClose} size="lg">
    <ModalHeader toggle={onClose}>
        {channel?.id
            ? $t("settings.notifications.channels.edit")
            : $t("settings.notifications.channels.add")}
    </ModalHeader>
    <ModalBody>
        <form id="channel-editor-form" onsubmit={(e) => { e.preventDefault(); save(); }}>
            <div class="mb-3">
                <label for="channel-type" class="form-label">
                    {$t("settings.notifications.channels.type")}
                </label>
                <Input
                    id="channel-type"
                    type="select"
                    bind:value={type}
                    on:change={onTypeChange}
                    disabled={!!channel?.id}
                >
                    {#each channelTypes as t (t)}
                        <option value={t}>{t}</option>
                    {/each}
                </Input>
                {#if channel?.id}
                    <small class="form-text text-muted">
                        {$t("settings.notifications.channels.typeImmutable")}
                    </small>
                {/if}
            </div>

            <div class="mb-3">
                <label for="channel-name" class="form-label">
                    {$t("settings.notifications.channels.name")}
                </label>
                <Input id="channel-name" type="text" bind:value={name} />
            </div>

            <div class="form-check form-switch mb-3">
                <input
                    class="form-check-input"
                    type="checkbox"
                    role="switch"
                    id="channel-enabled"
                    bind:checked={enabled}
                />
                <label class="form-check-label" for="channel-enabled">
                    {$t("settings.notifications.channels.enabled")}
                </label>
            </div>

            <hr />

            {#if schema}
                {#each schema.fields as field (field.key)}
                    <div class="mb-3">
                        <label for={`field-${field.key}`} class="form-label">
                            {$t(field.i18nLabel)}
                            {#if field.required}<span class="text-danger">*</span>{/if}
                        </label>

                        {#if field.kind === "text" || field.kind === "url"}
                            <Input
                                id={`field-${field.key}`}
                                type={field.kind === "url" ? "url" : "text"}
                                value={(config[field.key] as string) ?? ""}
                                on:input={(e) => {
                                    config = {
                                        ...config,
                                        [field.key]: (e.target as HTMLInputElement).value,
                                    };
                                }}
                                required={field.required}
                            />
                        {:else if field.kind === "secret"}
                            <div class="input-group">
                                <Input
                                    id={`field-${field.key}`}
                                    type={revealedSecrets[field.key] ? "text" : "password"}
                                    value={(config[field.key] as string) ?? ""}
                                    on:input={(e) => {
                                        config = {
                                            ...config,
                                            [field.key]: (e.target as HTMLInputElement).value,
                                        };
                                    }}
                                />
                                <Button
                                    type="button"
                                    color="secondary"
                                    outline
                                    on:click={() =>
                                        (revealedSecrets = {
                                            ...revealedSecrets,
                                            [field.key]: !revealedSecrets[field.key],
                                        })}
                                >
                                    <i
                                        class={`bi bi-eye${revealedSecrets[field.key] ? "-slash" : ""}`}
                                    ></i>
                                </Button>
                            </div>
                        {:else if field.kind === "headers"}
                            {@const headers = (config[field.key] as Record<string, string>) ?? {}}
                            {#each Object.entries(headers) as [k, v] (k)}
                                <div class="d-flex gap-2 mb-2">
                                    <Input
                                        type="text"
                                        placeholder={$t(
                                            "settings.notifications.channels.fields.headerName",
                                        )}
                                        value={k}
                                        on:change={(e) =>
                                            setHeaderKey(
                                                field,
                                                k,
                                                (e.target as HTMLInputElement).value,
                                            )}
                                    />
                                    <Input
                                        type="text"
                                        placeholder={$t(
                                            "settings.notifications.channels.fields.headerValue",
                                        )}
                                        value={v}
                                        on:input={(e) =>
                                            setHeaderValue(
                                                field,
                                                k,
                                                (e.target as HTMLInputElement).value,
                                            )}
                                    />
                                    <Button
                                        type="button"
                                        color="danger"
                                        outline
                                        on:click={() => removeHeader(field, k)}
                                    >
                                        <i class="bi bi-trash"></i>
                                    </Button>
                                </div>
                            {/each}
                            <Button
                                type="button"
                                size="sm"
                                color="secondary"
                                outline
                                on:click={() => addHeader(field)}
                            >
                                <i class="bi bi-plus-lg"></i>
                                {$t("settings.notifications.channels.fields.addHeader")}
                            </Button>
                        {/if}

                        {#if field.i18nHelp}
                            <small class="form-text text-muted d-block">
                                {$t(field.i18nHelp)}
                            </small>
                        {/if}
                    </div>
                {/each}
            {:else if type}
                <div class="mb-2">
                    <label for="raw-json" class="form-label">
                        {$t("settings.notifications.channels.rawJson")}
                    </label>
                    <textarea
                        id="raw-json"
                        class="form-control font-monospace"
                        rows="8"
                        bind:value={rawJson}
                    ></textarea>
                    {#if rawJsonError}
                        <div class="text-danger small">{rawJsonError}</div>
                    {/if}
                    <small class="form-text text-muted">
                        {$t("settings.notifications.channels.rawJsonHelp")}
                    </small>
                </div>
            {/if}
        </form>
    </ModalBody>
    <ModalFooter>
        <Button type="button" color="secondary" outline on:click={onClose} disabled={saving}>
            {$t("common.cancel")}
        </Button>
        <Button
            type="submit"
            form="channel-editor-form"
            color="primary"
            disabled={saving || !type}
        >
            {#if saving}<Spinner size="sm" class="me-2" />{/if}
            {$t("common.save")}
        </Button>
    </ModalFooter>
</Modal>
