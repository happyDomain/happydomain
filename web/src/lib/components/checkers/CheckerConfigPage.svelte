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
    import { Alert, Button, Card, Col, Icon, Row } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import { checkers } from "$lib/stores/checkers";
    import { toasts } from "$lib/stores/toasts";
    import type {
        HappydnsCheckPlan,
        HappydnsCheckPlanWritable,
        HappydnsCheckerOptionsPositional,
    } from "$lib/api-base/types.gen";
    import type { CheckerScope } from "$lib/api/checkers";
    import {
        getCheckStatus,
        getScopedCheckOptions,
        updateScopedCheckOptions,
    } from "$lib/api/checkers";
    import { splitPositionalOptions } from "$lib/utils";
    import PageTitle from "$lib/components/PageTitle.svelte";
    import CheckerScheduleCard from "./CheckerScheduleCard.svelte";
    import CheckerRulesCard from "./CheckerRulesCard.svelte";
    import CheckerOptionsPanel from "./CheckerOptionsPanel.svelte";

    interface Props {
        scope: CheckerScope;
        checksBase: string;
        checkerId: string;
        domainName: string;
        editableGroups: (status: any) => { label: string; opts: any[] }[];
        readOnlyGroups: (status: any) => { key: string; label: string; opts: any[] }[];
        showSchedule?: boolean;
    }

    let { scope, checksBase, checkerId, domainName, editableGroups, readOnlyGroups, showSchedule = true }: Props = $props();

    let checkStatusPromise = $derived(getCheckStatus(checkerId));
    let checkOptionsPromise = $derived(getScopedCheckOptions(scope, checkerId));

    let resolvedStatus = $state<any>(null);
    let optionValues = $state<Record<string, unknown>>({});
    let inheritedValues = $state<Record<string, unknown>>({});
    let savingOptions = $state(false);

    let checkerDef = $derived($checkers?.[checkerId]);
    let intervalSpec = $derived(checkerDef?.interval);

    let plan = $state<HappydnsCheckPlanWritable>({
        enabled: {},
    });

    $effect(() => {
        checkStatusPromise.then((status) => {
            resolvedStatus = status;
            if (status?.rules && Object.keys(plan.enabled ?? {}).length === 0) {
                const enabled: Record<string, boolean> = {};
                for (const rule of status.rules) {
                    if (rule.name) enabled[rule.name] = true;
                }
                plan.enabled = enabled;
            }
        });
    });

    $effect(() => {
        checkOptionsPromise.then((positionals: HappydnsCheckerOptionsPositional[]) => {
            const { current, inherited } = splitPositionalOptions(positionals);
            optionValues = current;
            inheritedValues = inherited;
        });
    });

    async function saveOptions() {
        savingOptions = true;
        try {
            await updateScopedCheckOptions(scope, checkerId, optionValues);
            checkOptionsPromise = getScopedCheckOptions(scope, checkerId);
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
        } finally {
            savingOptions = false;
        }
    }
</script>

<svelte:head>
    <title>{resolvedStatus?.name ?? checkerId} - {domainName} - happyDomain</title>
</svelte:head>

<div class="flex-fill mt-1 mb-5">
    <PageTitle title={resolvedStatus?.name ?? checkerId} domain={domainName}>
        {#if $checkers && (!$checkers[checkerId]?.availability || $checkers[checkerId].availability.applyToDomain || $checkers[checkerId].availability.applyToZone)}
            <Button
                color="info"
                href={`${checksBase}/${encodeURIComponent(checkerId)}/executions`}
            >
                <Icon name="bar-chart-fill"></Icon>
                {$t("checkers.list.view-results")}
            </Button>
        {/if}
    </PageTitle>

    {#await checkStatusPromise}
        <Card body>
            <p class="text-center mb-0">
                <span class="spinner-border spinner-border-sm me-2"></span>
                {$t("checkers.loading-info")}
            </p>
        </Card>
    {:then status}
        {#if status}
            {@const editable = editableGroups(status)}
            {@const readOnly = readOnlyGroups(status)}
            <Row class="mb-4">
                {#if showSchedule}
                <Col md={6}>
                    <CheckerScheduleCard {scope} {checkerId} bind:plan {intervalSpec} />

                    {#if status.rules && status.rules.length > 0}
                        <CheckerRulesCard
                            rules={status.rules}
                            bind:optionValues
                            {inheritedValues}
                            saving={savingOptions}
                            onsave={saveOptions}
                            bind:plan
                        />
                    {/if}
                </Col>
                {/if}

                <Col md={6}>
                    <CheckerOptionsPanel
                        {checkOptionsPromise}
                        editableGroups={editable}
                        readOnlyGroups={readOnly}
                        bind:optionValues
                        {inheritedValues}
                        saving={savingOptions}
                        onsave={saveOptions}
                    />
                </Col>
            </Row>
        {:else}
            <Alert color="danger">
                <Icon name="exclamation-triangle-fill"></Icon>
                {$t("checkers.checker-info-not-found")}
            </Alert>
        {/if}
    {:catch error}
        <Alert color="danger">
            <Icon name="exclamation-triangle-fill"></Icon>
            {$t("checkers.error-loading-checker", { error: error.message })}
        </Alert>
    {/await}
</div>
