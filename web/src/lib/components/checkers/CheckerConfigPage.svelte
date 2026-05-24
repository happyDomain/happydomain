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
    import { Alert, Badge, Button, Card, CardBody, CardHeader, Col, Icon, Row } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import { base } from "$lib/stores/config";
    import { checkers } from "$lib/stores/checkers";
    import { toasts } from "$lib/stores/toasts";
    import type {
        CheckerCheckerOptionDocumentation,
        HappydnsCheckPlan,
        HappydnsCheckPlanWritable,
        HappydnsCheckerOptionsPositional,
    } from "$lib/api-base/types.gen";
    import type { CheckerScope } from "$lib/api/checkers";
    import {
        getScopedCheckOptions,
        updateScopedCheckOptions,
        getScopedCheckStatus,
    } from "$lib/api/checkers";
    import { splitPositionalOptions, collectAutoFillKeys, collectAllOptionDocs, getOrphanedOptionKeys, filterValidOptions, availabilityBadges } from "$lib/utils";
    import PageTitle from "$lib/components/PageTitle.svelte";
    import CheckerScheduleCard from "./CheckerScheduleCard.svelte";
    import CheckerRulesCard from "./CheckerRulesCard.svelte";
    import CheckerOptionsPanel from "./CheckerOptionsPanel.svelte";
    import PrometheusMetricsModal from "./PrometheusMetricsModal.svelte";

    interface Props {
        scope: CheckerScope;
        checksBase: string;
        checkerId: string;
        domainName: string;
        groups: (status: any) => { editableGroups: { label: string; opts: any[] }[]; readOnlyGroups: { key: string; label: string; opts: any[] }[] };
        showSchedule?: boolean;
        showCheckerInfo?: boolean;
        showExecutions?: boolean;
    }

    let { scope, checksBase, checkerId, domainName, groups, showSchedule = true, showCheckerInfo = false, showExecutions = true }: Props = $props();

    let checkStatusPromise = $derived(getScopedCheckStatus(scope, checkerId));
    let checkOptionsPromise = $derived(getScopedCheckOptions(scope, checkerId));

    let resolvedStatus = $state<any>(null);
    let optionValues = $state<Record<string, unknown>>({});
    let inheritedValues = $state<Record<string, unknown>>({});
    let savingOptions = $state(false);

    let checkerDef = $derived($checkers?.[checkerId]);
    let intervalSpec = $derived(checkerDef?.interval);
    let metricsApiUrl = $derived(
        scope.zoneId && scope.subdomain !== undefined && scope.serviceId
            ? `${base}/api/domains/${scope.domainId}/zone/${scope.zoneId}/${scope.subdomain}/services/${scope.serviceId}/checkers/${encodeURIComponent(checkerId)}/metrics`
            : `${base}/api/domains/${scope.domainId}/checkers/${encodeURIComponent(checkerId)}/metrics`
    );

    let plan = $state<HappydnsCheckPlanWritable>({
        enabled: {},
    });
    let scheduleCard = $state<{ save: () => Promise<void> } | undefined>(undefined);
    let metricsModalOpen = $state(false);

    $effect(() => {
        // Reset state when switching checkers
        checkerId;
        plan = { enabled: {} };
        resolvedStatus = null;
        optionValues = {};
        inheritedValues = {};
    });

    $effect(() => {
        checkStatusPromise.then((status) => {
            resolvedStatus = status;
            if (status?.rules) {
                const enabled: Record<string, boolean> = {};
                for (const rule of status.rules) {
                    if (rule.name) enabled[rule.name] = true;
                }
                plan.enabled = enabled;
            }
        });
    });

    // Returns true when a positional belongs to the current page's scope.
    // A positional with no domainId is admin-scope; one with domainId but no
    // serviceId is domain-scope; one with both is service-scope.
    function isCurrentScopePositional(p: { domainId?: unknown; serviceId?: unknown }): boolean {
        const hasDomain = Array.isArray(p.domainId) ? p.domainId.length > 0 : !!p.domainId;
        const hasService = Array.isArray(p.serviceId) ? p.serviceId.length > 0 : !!p.serviceId;
        if (!scope.domainId) return !hasDomain;
        if (!scope.serviceId) return hasDomain && !hasService;
        return hasDomain && hasService;
    }

    $effect(() => {
        Promise.all([checkStatusPromise, checkOptionsPromise]).then(
            ([status, positionals]: [any, HappydnsCheckerOptionsPositional[]]) => {
                const autoFillKeys = status ? collectAutoFillKeys(status) : new Set<string>();
                const { current, inherited } = splitPositionalOptions(positionals, autoFillKeys, isCurrentScopePositional);
                optionValues = current;
                inheritedValues = inherited;
            },
        );
    });

    async function saveOptions() {
        savingOptions = true;
        try {
            await updateScopedCheckOptions(scope, checkerId, optionValues);
            checkOptionsPromise = getScopedCheckOptions(scope, checkerId);
        } finally {
            savingOptions = false;
        }
    }

    async function cleanOrphanedOptions(allEditableOpts: CheckerCheckerOptionDocumentation[]) {
        savingOptions = true;
        try {
            await updateScopedCheckOptions(scope, checkerId, filterValidOptions(optionValues, allEditableOpts));
            checkOptionsPromise = getScopedCheckOptions(scope, checkerId);
            toasts.addToast({
                message: $t("checkers.messages.options-cleaned"),
                type: "success",
                timeout: 5000,
            });
        } catch (error) {
            toasts.addErrorToast({
                message: $t("checkers.messages.clean-failed", { error: String(error) }),
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
        {#if showExecutions && $checkers && (!$checkers[checkerId]?.availability || $checkers[checkerId].availability.applyToDomain || $checkers[checkerId].availability.applyToZone)}
            <Button
                color="info"
                href={`${checksBase}/${encodeURIComponent(checkerId)}/executions`}
            >
                <Icon name="bar-chart-fill"></Icon>
                {$t("checkers.list.view-results")}
            </Button>
        {/if}
        {#if scope.domainId && checkerDef?.has_metrics}
            <Button
                color="secondary"
                outline
                onclick={() => (metricsModalOpen = true)}
            >
                <Icon name="graph-up-arrow"></Icon>
                {$t("checkers.list.prometheus-metrics")}
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
            {@const { editableGroups: editable, readOnlyGroups: readOnly } = groups(status)}
            {@const allEditableOpts = collectAllOptionDocs(status)}
            {@const orphanedOpts = getOrphanedOptionKeys(optionValues, allEditableOpts)}
            {@const hasLeftCol = showSchedule || showCheckerInfo || !!(status.rules && status.rules.length > 0)}
            <Row class="mb-4">
                {#if hasLeftCol}
                <Col md={6}>
                    {#if showCheckerInfo}
                        <Card class="mb-3">
                            <CardHeader>
                                <strong>{$t("checkers.detail.checker-information")}</strong>
                            </CardHeader>
                            <CardBody>
                                <dl class="row mb-0">
                                    <dt class="col-sm-4">{$t("checkers.detail.name")}</dt>
                                    <dd class="col-sm-8">{status.name}</dd>

                                    <dt class="col-sm-4">{$t("checkers.detail.availability")}</dt>
                                    <dd class="col-sm-8">
                                        {#each availabilityBadges(status.availability, $t) as badge}
                                            <Badge color={badge.color}>{badge.label}</Badge>
                                        {:else}
                                            <Badge color="secondary">
                                                {$t("checkers.availability.general")}
                                            </Badge>
                                        {/each}
                                    </dd>
                                </dl>
                            </CardBody>
                        </Card>
                    {/if}

                    {#if showSchedule}
                        <CheckerScheduleCard bind:this={scheduleCard} {scope} {checkerId} bind:plan {intervalSpec} />
                    {/if}

                    {#if status.rules && status.rules.length > 0}
                        <CheckerRulesCard
                            rules={status.rules}
                            bind:optionValues
                            {inheritedValues}
                            saving={savingOptions}
                            onsave={saveOptions}
                            onsaveplan={showSchedule ? () => scheduleCard!.save() : undefined}
                            bind:plan
                            precheckFailures={(status as any).precheckFailures}
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
                        {orphanedOpts}
                        onclean={() => cleanOrphanedOptions(allEditableOpts)}
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

<PrometheusMetricsModal bind:isOpen={metricsModalOpen} url={metricsApiUrl} />
