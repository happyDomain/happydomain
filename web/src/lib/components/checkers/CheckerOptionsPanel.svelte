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
    import {
        Alert,
        Badge,
        Button,
        Card,
        CardBody,
        CardHeader,
        Form,
        Icon,
    } from "@sveltestrap/sveltestrap";
    import type {
        CheckerCheckerOptionDocumentation,
        HappydnsCheckerOptionsPositional,
    } from "$lib/api-base/types.gen";
    import { t } from "$lib/translations";
    import { toasts } from "$lib/stores/toasts";
    import { withInheritedPlaceholders } from "$lib/utils/checkers";
    import ResourceInput from "$lib/components/inputs/Resource.svelte";
    import CheckerOptionsGroups from "./CheckerOptionsGroups.svelte";

    interface EditableGroup {
        label: string;
        opts: CheckerCheckerOptionDocumentation[];
    }

    interface ReadOnlyGroup {
        key: string;
        label: string;
        opts: CheckerCheckerOptionDocumentation[];
    }

    interface Props {
        checkOptionsPromise: Promise<HappydnsCheckerOptionsPositional[]>;
        editableGroups: EditableGroup[];
        readOnlyGroups: ReadOnlyGroup[];
        optionValues: Record<string, unknown>;
        inheritedValues: Record<string, unknown>;
        saving: boolean;
        onsave: () => Promise<void>;
        orphanedOpts?: string[];
        onclean?: () => void;
    }

    let {
        checkOptionsPromise,
        editableGroups,
        readOnlyGroups,
        optionValues = $bindable(),
        inheritedValues,
        saving,
        onsave,
        orphanedOpts = [],
        onclean,
    }: Props = $props();

    // Filter out auto-fill fields (system-provided, never user-edited) and
    // noOverride fields (locked by a broader scope; nothing to do here).
    let filteredEditableGroups = $derived(
        editableGroups.map((g) => ({
            ...g,
            opts: withInheritedPlaceholders(
                g.opts.filter((opt) => !opt.autoFill && !opt.noOverride),
                optionValues,
                inheritedValues,
            ),
        })),
    );

    // Collect auto-fill fields into read-only groups for display.
    let autoFillOpts = $derived(
        editableGroups.flatMap((g) => g.opts.filter((opt) => opt.autoFill)),
    );

    let hasAnyOpts = $derived(
        filteredEditableGroups.some((g) => g.opts.length > 0) ||
            readOnlyGroups.some((g) => g.opts.length > 0) ||
            autoFillOpts.length > 0,
    );

    async function handleSave() {
        try {
            await onsave();
            toasts.addToast({
                message: $t("checkers.messages.options-updated"),
                type: "success",
                timeout: 5000,
            });
        } catch (error) {
            toasts.addErrorToast({
                message: $t("checkers.messages.update-failed", { error: String(error) }),
                timeout: 10000,
            });
        }
    }
</script>

{#await checkOptionsPromise}
    <Card>
        <CardBody>
            <p class="text-center mb-0">
                <span class="spinner-border spinner-border-sm me-2"></span>
                {$t("checkers.detail.loading-options")}
            </p>
        </CardBody>
    </Card>
{:then _options}
    {#if orphanedOpts.length > 0 && onclean}
        <Alert color="warning" class="mb-3">
            <div class="d-flex justify-content-between align-items-center">
                <div>
                    <Icon name="exclamation-triangle-fill"></Icon>
                    {$t("checkers.detail.orphaned-options", {
                        options: orphanedOpts.join(", "),
                    })}
                </div>
                <Button type="button" color="danger" size="sm" onclick={onclean} disabled={saving}>
                    <Icon name="trash"></Icon>
                    {$t("checkers.detail.clean-up")}
                </Button>
            </div>
        </Alert>
    {/if}

    {#each filteredEditableGroups.filter((g) => g.opts.length > 0) as group, gid}
        <Card class="mb-3">
            <CardHeader class="d-flex align-items-center justify-content-between">
                <strong>{group.label}</strong>
                <Button
                    color="success"
                    form={"group-" + gid}
                    size="sm"
                    onclick={handleSave}
                    disabled={saving}
                >
                    {#if saving}
                        <span class="spinner-border spinner-border-sm me-1"></span>
                    {:else}
                        <Icon name="check-circle"></Icon>
                    {/if}
                    {$t("checkers.detail.save")}
                </Button>
            </CardHeader>
            <CardBody>
                <Form id={"group-" + gid} onsubmit={handleSave}>
                    {#each group.opts as optDoc, index}
                        {#if optDoc.id}
                            {@const optId = optDoc.id}
                            {@const inherited = inheritedValues[optId]}
                            {@const localVal = optionValues[optId]}
                            {@const isOverriding =
                                localVal !== undefined &&
                                localVal !== "" &&
                                inherited !== undefined}
                            <div class="option-row mb-2">
                                <ResourceInput
                                    edit
                                    index={"" + index}
                                    specs={optDoc}
                                    type={optDoc.type || "string"}
                                    bind:value={optionValues[optId]}
                                />
                                {#if isOverriding}
                                    <div class="form-text mt-1 d-flex align-items-center gap-2 flex-wrap">
                                        <Badge color="warning">
                                            <Icon name="pencil-square"></Icon>
                                            {$t("checkers.detail.overriding-inherited")}
                                        </Badge>
                                        <span class="text-muted">
                                            {$t("checkers.detail.inherited-value", { value: String(inherited) })}
                                        </span>
                                        <Button
                                            color="link"
                                            size="sm"
                                            class="p-0"
                                            onclick={() => {
                                                const next = { ...optionValues };
                                                delete next[optId];
                                                optionValues = next;
                                            }}
                                        >
                                            {$t("checkers.detail.reset-to-inherited")}
                                        </Button>
                                    </div>
                                {/if}
                            </div>
                        {/if}
                    {/each}
                </Form>
            </CardBody>
        </Card>
    {/each}

    {#if autoFillOpts.length > 0}
        <CheckerOptionsGroups
            groups={[
                {
                    key: "auto-fill",
                    label: $t("checkers.detail.auto-fill"),
                    opts: autoFillOpts,
                },
            ]}
            optionValues={inheritedValues}
        />
    {/if}

    <CheckerOptionsGroups groups={readOnlyGroups} optionValues={inheritedValues} />

    {#if !hasAnyOpts}
        <Card>
            <CardBody>
                <Alert color="info" class="mb-0">
                    <Icon name="info-circle"></Icon>
                    {$t("checkers.detail.no-configurable-options")}
                </Alert>
            </CardBody>
        </Card>
    {/if}
{:catch error}
    <Card>
        <CardBody>
            <Alert color="danger" class="mb-0">
                <Icon name="exclamation-triangle-fill"></Icon>
                {$t("checkers.detail.error-loading-options", {
                    error: error.message,
                })}
            </Alert>
        </CardBody>
    </Card>
{/await}
