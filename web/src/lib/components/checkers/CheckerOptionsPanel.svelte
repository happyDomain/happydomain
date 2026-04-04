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
        onsave: () => void;
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

    // Filter out auto-fill fields from editable groups (they are system-provided).
    let filteredEditableGroups = $derived(
        editableGroups.map((g) => ({
            ...g,
            opts: g.opts.filter((opt) => !opt.autoFill),
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
                    onclick={onsave}
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
                <Form id={"group-" + gid} onsubmit={onsave}>
                    {#each withInheritedPlaceholders(group.opts, optionValues, inheritedValues) as optDoc, index}
                        {#if optDoc.id}
                            <ResourceInput
                                edit
                                index={"" + index}
                                specs={optDoc}
                                type={optDoc.type || "string"}
                                bind:value={optionValues[optDoc.id]}
                            />
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
        />
    {/if}

    <CheckerOptionsGroups groups={readOnlyGroups} />

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
