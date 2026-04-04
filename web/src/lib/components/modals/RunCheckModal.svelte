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
        Form,
        FormGroup,
        Icon,
        Input,
        Label,
        Modal,
        ModalBody,
        ModalFooter,
        ModalHeader,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { getCheckStatus, getScopedCheckOptions, triggerScopedCheck } from "$lib/api/checkers";
    import type { CheckerScope } from "$lib/api/checkers";
    import { collectAllOptionDocs } from "$lib/utils/checkers";
    import type {
        CheckerCheckerDefinition,
        CheckerCheckerOptionDocumentation,
        CheckerCheckRuleInfo,
        HappydnsCheckerOptions,
        HappydnsCheckerOptionsPositional,
        HappydnsCheckerRunRequest,
    } from "$lib/api-base/types.gen";
    import Resource from "$lib/components/inputs/Resource.svelte";
    import { toasts } from "$lib/stores/toasts";
    import { t } from "$lib/translations";

    interface Props {
        scope: CheckerScope;
        onCheckTriggered?: (execution_id: string) => void;
    }

    let { scope, onCheckTriggered }: Props = $props();

    let isOpen = $state(false);
    let checkName = $state<string>("");
    let checkDisplayName = $state<string>("");
    let checkStatusPromise = $state<Promise<CheckerCheckerDefinition> | null>(null);
    let scopedOptionsPromise = $state<Promise<HappydnsCheckerOptionsPositional[]> | null>(null);
    let resolvedStatus = $state<CheckerCheckerDefinition | null>(null);
    let runOptions = $state<Record<string, unknown>>({});
    let scopedDefaults = $state<Record<string, unknown>>({});
    let triggering = $state(false);
    let showAdvanced = $state(false);
    let activeRules = $state<Record<number, boolean>>({});

    const toggle = () => (isOpen = !isOpen);

    export function open(name: string, displayName: string) {
        checkName = name;
        checkDisplayName = displayName;
        runOptions = {};
        scopedDefaults = {};
        showAdvanced = false;
        activeRules = {};
        resolvedStatus = null;
        checkStatusPromise = getCheckStatus(name);
        scopedOptionsPromise = getScopedCheckOptions(scope, name);
        isOpen = true;

        Promise.all([checkStatusPromise, scopedOptionsPromise]).then(
            ([status, options]: [
                CheckerCheckerDefinition,
                HappydnsCheckerOptionsPositional[],
            ]) => {
                resolvedStatus = status;
                scopedDefaults = Object.assign({}, ...(options || []).map((p) => p.options || {}));

                // For select fields (choices), set the value directly since placeholders don't work on <select>.
                const allOpts = collectAllOptionDocs(status);
                for (const opt of allOpts) {
                    if (opt.id && opt.choices?.length && opt.id in scopedDefaults) {
                        runOptions[opt.id] = scopedDefaults[opt.id];
                    }
                }
            },
        );
    }

    function getActiveOptionIds(): Set<string> {
        const ids = new Set<string>();
        if (!resolvedStatus) return ids;
        const addOpts = (opts: CheckerCheckerOptionDocumentation[] | undefined) =>
            opts?.forEach((o) => o.id && !o.noOverride && ids.add(o.id));
        addOpts(resolvedStatus.options?.runOpts);
        addOpts(resolvedStatus.options?.adminOpts);
        addOpts(resolvedStatus.options?.userOpts);
        addOpts(resolvedStatus.options?.domainOpts);
        resolvedStatus.rules?.forEach((rule: CheckerCheckRuleInfo, idx: number) => {
            if (activeRules[idx] !== false) {
                if (rule.options) {
                    addOpts(rule.options.runOpts);
                    addOpts(rule.options.adminOpts);
                    addOpts(rule.options.userOpts);
                    addOpts(rule.options.domainOpts);
                }
            }
        });
        return ids;
    }

    function specsWithPlaceholder(
        optDoc: CheckerCheckerOptionDocumentation,
    ): CheckerCheckerOptionDocumentation {
        if (optDoc.id && optDoc.id in scopedDefaults && scopedDefaults[optDoc.id] != null) {
            return { ...optDoc, placeholder: String(scopedDefaults[optDoc.id]) };
        }
        return optDoc;
    }

    async function handleRunCheck() {
        triggering = true;
        try {
            const activeIds = getActiveOptionIds();
            const filteredOptions: HappydnsCheckerOptions = {};
            for (const [k, v] of Object.entries(runOptions)) {
                if (!resolvedStatus || activeIds.has(k)) filteredOptions[k] = v;
            }

            // Build enabledRules map from activeRules (only if some rules are toggled off).
            const rules = resolvedStatus?.rules ?? [];
            let enabledRules: Record<string, boolean> | undefined;
            if (rules.length > 0) {
                const hasDisabled = rules.some((_r: CheckerCheckRuleInfo, idx: number) => activeRules[idx] === false);
                if (hasDisabled) {
                    enabledRules = {};
                    for (let i = 0; i < rules.length; i++) {
                        const name = rules[i].name;
                        if (name) {
                            enabledRules[name] = activeRules[i] !== false;
                        }
                    }
                }
            }

            const request: HappydnsCheckerRunRequest = {
                options: filteredOptions,
                ...(enabledRules ? { enabledRules } : {}),
            };
            const result = await triggerScopedCheck(scope, checkName, request);
            toasts.addToast({
                message: $t("checkers.run-check.triggered-success", { id: result.id ?? "" }),
                type: "success",
                timeout: 5000,
            });
            isOpen = false;
            if (onCheckTriggered && result.id) {
                onCheckTriggered(result.id);
            }
        } catch (error) {
            toasts.addErrorToast({
                message: $t("checkers.run-check.trigger-failed", { error: String(error) }),
                timeout: 10000,
            });
        } finally {
            triggering = false;
        }
    }
</script>

<Modal {isOpen} {toggle} size="lg">
    <ModalHeader {toggle}>
        {$t("checkers.run-check.title")}: {checkDisplayName}
    </ModalHeader>
    <ModalBody>
        {#if checkStatusPromise && scopedOptionsPromise}
            {#await Promise.all([checkStatusPromise, scopedOptionsPromise])}
                <div class="text-center py-3">
                    <Spinner />
                    <p class="mt-2">{$t("checkers.run-check.loading-options")}</p>
                </div>
            {:then [status, _domainOpts]}
                {@const rules = status.rules || []}
                {@const activeRulesForOpts = rules.map(
                    (r: CheckerCheckRuleInfo, i: number) =>
                        activeRules[i] !== false ? r : null,
                )}
                {@const runOpts = [
                    ...(status.options?.runOpts || []),
                    ...activeRulesForOpts.flatMap((r: CheckerCheckRuleInfo | null) => r?.options?.runOpts || []),
                ].filter((o: CheckerCheckerOptionDocumentation) => !o.noOverride)}
                {@const otherOpts = [
                    ...(status.options?.adminOpts || []),
                    ...(status.options?.userOpts || []),
                    ...(status.options?.domainOpts || []),
                    ...activeRulesForOpts.flatMap((r: CheckerCheckRuleInfo | null) => [
                        ...(r?.options?.adminOpts || []),
                        ...(r?.options?.userOpts || []),
                        ...(r?.options?.domainOpts || []),
                    ]),
                ].filter((o: CheckerCheckerOptionDocumentation) => o.id && !o.noOverride)}
                <Form
                    id="run-check-modal"
                    onsubmit={(e: Event) => {
                        e.preventDefault();
                        handleRunCheck();
                    }}
                >
                    {#if runOpts.length > 0 || otherOpts.length > 0}
                        <p>
                            {#if runOpts.length > 0}
                                {$t("checkers.run-check.configure-info")}
                            {:else}
                                <Icon name="info-circle"></Icon>
                                {$t("checkers.run-check.no-run-options")}
                            {/if}
                        </p>
                        {#each runOpts as optDoc}
                            {#if optDoc.id}
                                {@const optName = optDoc.id}
                                <FormGroup>
                                    <Resource
                                        edit={true}
                                        index={optName}
                                        specs={specsWithPlaceholder(optDoc)}
                                        type={optDoc.type || "string"}
                                        readonly={!!optDoc.autoFill}
                                        bind:value={runOptions[optName]}
                                    />
                                </FormGroup>
                            {/if}
                        {/each}
                        {#if otherOpts.length > 0}
                            <button
                                type="button"
                                class="btn btn-link btn-sm px-0 mb-2 text-muted d-flex align-items-center gap-1 text-decoration-none"
                                onclick={() => (showAdvanced = !showAdvanced)}
                            >
                                <Icon name={showAdvanced ? "chevron-down" : "chevron-right"} />
                                {$t("checkers.run-check.advanced-options")}
                            </button>
                            {#if showAdvanced}
                                {#each otherOpts as optDoc}
                                    {@const optName = optDoc.id}
                                    {#if optName}
                                    <FormGroup>
                                        <Resource
                                            edit={true}
                                            index={optName}
                                            specs={specsWithPlaceholder(optDoc)}
                                            type={optDoc.type || "string"}
                                            readonly={!!optDoc.autoFill}
                                            bind:value={runOptions[optName]}
                                        />
                                    </FormGroup>
                                    {/if}
                                {/each}
                            {/if}
                        {/if}
                    {:else}
                        <Alert color="info" class="mb-0">
                            <Icon name="info-circle"></Icon>
                            {$t("checkers.run-check.no-options")}
                        </Alert>
                    {/if}
                    {#if rules.length >= 1}
                        <hr />
                        <FormGroup>
                            <Label>{$t("checkers.run-check.rules")}</Label>
                            {#each rules as rule, idx}
                                {@const isActive = activeRules[idx] !== false}
                                <div class="form-check">
                                    <Input
                                        type="checkbox"
                                        id="run-check-rule-{idx}"
                                        label={rule.name ?? String(idx)}
                                        checked={isActive}
                                        onchange={() => (activeRules[idx] = !isActive)}
                                    />
                                </div>
                            {/each}
                        </FormGroup>
                    {/if}
                </Form>
            {:catch error}
                <Alert color="danger">
                    <Icon name="exclamation-triangle-fill"></Icon>
                    {$t("checkers.run-check.error-loading-options", { error: error.message })}
                </Alert>
            {/await}
        {/if}
    </ModalBody>
    <ModalFooter>
        <Button type="button" color="secondary" onclick={toggle} disabled={triggering}>
            {$t("common.cancel")}
        </Button>
        <Button
            type="submit"
            form="run-check-modal"
            color="primary"
            disabled={triggering}
        >
            {#if triggering}
                <Spinner size="sm" class="me-1" />
            {:else}
                <Icon name="play-fill"></Icon>
            {/if}
            {$t("checkers.run-check.run-button")}
        </Button>
    </ModalFooter>
</Modal>
