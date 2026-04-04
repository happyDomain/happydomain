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
        Badge,
        Button,
        Card,
        CardHeader,
        Form,
        Icon,
        ListGroup,
        ListGroupItem,
    } from "@sveltestrap/sveltestrap";
    import type { CheckerCheckRuleInfo, HappydnsCheckPlan, HappydnsCheckPlanWritable } from "$lib/api-base/types.gen";
    import { t } from "$lib/translations";
    import { withInheritedPlaceholders } from "$lib/utils/checkers";
    import ResourceInput from "$lib/components/inputs/Resource.svelte";

    interface Props {
        rules: CheckerCheckRuleInfo[];
        optionValues: Record<string, unknown>;
        inheritedValues: Record<string, unknown>;
        saving: boolean;
        onsave: () => void;
        plan?: HappydnsCheckPlan | HappydnsCheckPlanWritable;
    }

    let { rules, optionValues = $bindable(), inheritedValues, saving, onsave, plan = $bindable() }: Props = $props();

    let hasRuleOpts = $derived(
        rules.some((r) => (r.options?.adminOpts?.length ?? 0) + (r.options?.userOpts?.length ?? 0) > 0),
    );

    let allEnabled = $derived(
        plan && rules.length > 0 && rules.every((r) => r.name && plan!.enabled?.[r.name]),
    );

    function toggleAll() {
        if (!plan) return;
        const newVal = !allEnabled;
        const enabled: Record<string, boolean> = {};
        for (const rule of rules) {
            if (rule.name) enabled[rule.name] = newVal;
        }
        plan.enabled = enabled;
    }
</script>

<Card>
    <CardHeader class="d-flex align-items-center justify-content-between">
        <div>
            <strong>{$t("checkers.detail.check-rules")}</strong>
            <Badge color="secondary" class="ms-2">{rules.length}</Badge>
        </div>
        {#if plan}
            <div class="d-flex gap-2">
                <div class="form-check form-switch">
                    <input
                        class="form-check-input"
                        type="checkbox"
                        checked={allEnabled}
                        onchange={toggleAll}
                        id="toggle-all-rules"
                    />
                    <label class="form-check-label" for="toggle-all-rules">
                        All
                    </label>
                </div>
            </div>
        {:else if hasRuleOpts}
            <Button
                type="button"
                color="success"
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
        {/if}
    </CardHeader>
    <ListGroup flush>
        {#each rules as rule}
            {@const ruleOpts = [
                ...(rule.options?.adminOpts || []),
                ...(rule.options?.userOpts || []),
            ]}
            <ListGroupItem>
                <div class="d-flex align-items-start gap-2 mb-1">
                    {#if plan}
                        <div class="form-check form-switch mt-1">
                            <input
                                class="form-check-input"
                                type="checkbox"
                                checked={plan.enabled?.[rule.name ?? ""] ?? false}
                                onchange={() => {
                                    if (rule.name && plan) {
                                        plan.enabled = {
                                            ...plan.enabled,
                                            [rule.name]: !(plan.enabled?.[rule.name] ?? false),
                                        };
                                    }
                                }}
                            />
                        </div>
                    {:else}
                        <Icon
                            name="check2-circle"
                            class="text-success mt-1 flex-shrink-0"
                        ></Icon>
                    {/if}
                    <div class="flex-grow-1">
                        <strong>{rule.name}</strong>
                        {#if rule.description}
                            <p class="text-muted small mb-0">{rule.description}</p>
                        {/if}
                    </div>
                </div>
                {#if ruleOpts.length > 0}
                    <div class="ms-4 mt-2">
                        <Form onsubmit={onsave}>
                            {#each withInheritedPlaceholders(ruleOpts, optionValues, inheritedValues) as optDoc, index}
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
                    </div>
                {/if}
            </ListGroupItem>
        {/each}
    </ListGroup>
</Card>
