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
        Modal,
        ModalBody,
        ModalFooter,
        ModalHeader,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { triggerCheck, getDomainCheckOptions, getCheckStatus } from "$lib/api/checks";
    import type { CheckerOptions } from "$lib/model/check";
    import Resource from "$lib/components/inputs/Resource.svelte";
    import { toasts } from "$lib/stores/toasts";
    import { t } from "$lib/translations";

    interface Props {
        domainId: string;
        onCheckTriggered?: (execution_id: string, checker_name: string) => void;
    }

    let { domainId, onCheckTriggered }: Props = $props();

    let isOpen = $state(false);
    let checkName = $state<string>("");
    let checkDisplayName = $state<string>("");
    let checkStatusPromise = $state<Promise<any> | null>(null);
    let domainOptionsPromise = $state<Promise<CheckerOptions> | null>(null);
    let runOptions = $state<Record<string, any>>({});
    let triggering = $state(false);
    let showAdvanced = $state(false);

    const toggle = () => (isOpen = !isOpen);

    export function open(name: string, displayName: string) {
        checkName = name;
        checkDisplayName = displayName;
        runOptions = {};
        checkStatusPromise = getCheckStatus(name);
        domainOptionsPromise = getDomainCheckOptions(domainId, name);
        isOpen = true;

        // Pre-populate with domain options when they load
        domainOptionsPromise.then((options) => {
            runOptions = { ...(options || {}) };
        });
    }

    async function handleRunCheck() {
        triggering = true;
        try {
            const result = await triggerCheck(domainId, checkName, runOptions);
            toasts.addToast({
                message: $t("checks.run-check.triggered-success", { id: result.execution_id }),
                type: "success",
                timeout: 5000,
            });
            isOpen = false;
            if (onCheckTriggered && result.execution_id) {
                onCheckTriggered(result.execution_id, checkName);
            }
        } catch (error) {
            toasts.addErrorToast({
                message: $t("checks.run-check.trigger-failed", { error: String(error) }),
                timeout: 10000,
            });
        } finally {
            triggering = false;
        }
    }
</script>

<Modal {isOpen} {toggle} size="lg">
    <ModalHeader {toggle}>
        {$t("checks.run-check.title")}: {checkDisplayName}
    </ModalHeader>
    <ModalBody>
        {#if checkStatusPromise && domainOptionsPromise}
            {#await Promise.all([checkStatusPromise, domainOptionsPromise])}
                <div class="text-center py-3">
                    <Spinner />
                    <p class="mt-2">{$t("checks.run-check.loading-options")}</p>
                </div>
            {:then [status, _domainOpts]}
                {@const runOpts = status.options?.runOpts || []}
                {#if runOpts.length > 0}
                    <p>
                        {$t("checks.run-check.configure-info")}
                    </p>
                    <Form
                        id="run-test-modal"
                        on:submit={(e) => {
                            e.preventDefault();
                            handleRunCheck();
                        }}
                    >
                        {#each runOpts as optDoc}
                            {#if optDoc.id}
                                {@const optName = optDoc.id}
                                <FormGroup>
                                    <Resource
                                        edit={true}
                                        index={optName}
                                        specs={optDoc}
                                        type={optDoc.type || "string"}
                                        readonly={!!optDoc.autoFill}
                                        bind:value={runOptions[optName]}
                                    />
                                </FormGroup>
                            {/if}
                        {/each}
                        {@const otherOpts = [
                            ...(status.options?.adminOpts || []),
                            ...(status.options?.userOpts || []),
                            ...(status.options?.domainOpts || []),
                            ...(status.options?.serviceOpts || []),
                        ].filter((o) => o.id)}
                        {#if otherOpts.length > 0}
                            <button
                                type="button"
                                class="btn btn-link btn-sm px-0 mb-2 text-muted d-flex align-items-center gap-1 text-decoration-none"
                                onclick={() => (showAdvanced = !showAdvanced)}
                            >
                                <Icon name={showAdvanced ? "chevron-down" : "chevron-right"} />
                                {$t("checks.run-check.advanced-options")}
                            </button>
                            {#if showAdvanced}
                                {#each otherOpts as optDoc}
                                    {@const optName = optDoc.id}
                                    <FormGroup>
                                        <Resource
                                            edit={true}
                                            index={optName}
                                            specs={optDoc}
                                            type={optDoc.type || "string"}
                                            readonly={true}
                                            bind:value={runOptions[optName]}
                                        />
                                    </FormGroup>
                                {/each}
                            {/if}
                        {/if}
                    </Form>
                {:else}
                    <Alert color="info" class="mb-0">
                        <Icon name="info-circle"></Icon>
                        {$t("checks.run-check.no-options")}
                    </Alert>
                {/if}
            {:catch error}
                <Alert color="danger">
                    <Icon name="exclamation-triangle-fill"></Icon>
                    {$t("checks.run-check.error-loading-options", { error: error.message })}
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
            form="run-test-modal"
            color="primary"
            onclick={handleRunCheck}
            disabled={triggering}
        >
            {#if triggering}
                <Spinner size="sm" class="me-1" />
            {:else}
                <Icon name="play-fill"></Icon>
            {/if}
            {$t("checks.run-check.run-button")}
        </Button>
    </ModalFooter>
</Modal>
