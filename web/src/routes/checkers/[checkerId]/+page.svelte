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
        Card,
        CardBody,
        CardHeader,
        Col,
        Icon,
        Row,
    } from "@sveltestrap/sveltestrap";
    import { page } from "$app/state";

    import { t } from "$lib/translations";
    import { toasts } from "$lib/stores/toasts";
    import PageTitle from "$lib/components/PageTitle.svelte";
    import { getCheckStatus, getCheckOptions, updateCheckOptions } from "$lib/api/checkers";
    import type { CheckerCheckerOptionDocumentation, CheckerCheckRuleInfo, HappydnsCheckerOptionsPositional } from "$lib/api-base/types.gen";
    import CheckerRulesCard from "$lib/components/checkers/CheckerRulesCard.svelte";
    import CheckerOptionsPanel from "$lib/components/checkers/CheckerOptionsPanel.svelte";
    import { availabilityBadges, splitPositionalOptions, getOrphanedOptionKeys, filterValidOptions } from "$lib/utils";

    let checkerId = $derived(page.params.checkerId!);

    let checkStatusPromise = $derived(getCheckStatus(checkerId));
    let checkOptionsPromise = $derived(getCheckOptions(checkerId));
    let optionValues = $state<Record<string, unknown>>({});
    let inheritedValues = $state<Record<string, unknown>>({});
    let resolvedStatus = $state<any>(null);
    let saving = $state(false);

    $effect(() => {
        checkStatusPromise.then((status) => {
            resolvedStatus = status;
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
        saving = true;
        try {
            await updateCheckOptions(checkerId, optionValues);
            checkOptionsPromise = getCheckOptions(checkerId);
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
            saving = false;
        }
    }

    async function cleanOrphanedOptions(allEditableOpts: CheckerCheckerOptionDocumentation[]) {
        saving = true;
        try {
            await updateCheckOptions(checkerId, filterValidOptions(optionValues, allEditableOpts));
            checkOptionsPromise = getCheckOptions(checkerId);
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
            saving = false;
        }
    }

    function getOrphanedOptions(
        allEditableOpts: CheckerCheckerOptionDocumentation[],
        readOnlyGroups: { opts: CheckerCheckerOptionDocumentation[] }[],
    ): string[] {
        const allKnownOpts = [...allEditableOpts, ...readOnlyGroups.flatMap((g) => g.opts)];
        return getOrphanedOptionKeys(optionValues, allKnownOpts);
    }
</script>

<svelte:head>
    <title>{resolvedStatus?.name ?? checkerId} - {$t("checkers.title")} - happyDomain</title>
</svelte:head>

<div class="flex-fill mt-1 mb-5">
    <PageTitle title={resolvedStatus?.name ?? checkerId}></PageTitle>

    {#await checkStatusPromise}
        <Card body>
            <p class="text-center mb-0">
                <span class="spinner-border spinner-border-sm me-2"></span>
                {$t("checkers.loading-info")}
            </p>
        </Card>
    {:then status}
        {#if status}
            {@const adminOpts = status.options?.adminOpts || []}
            {@const userOpts = status.options?.userOpts || []}
            {@const rulesAdminOpts = (status.rules || []).flatMap((r: CheckerCheckRuleInfo) => r.options?.adminOpts || [])}
            {@const rulesUserOpts = (status.rules || []).flatMap((r: CheckerCheckRuleInfo) => r.options?.userOpts || [])}
            {@const allEditableOpts = [...adminOpts, ...userOpts, ...rulesAdminOpts, ...rulesUserOpts]}
            {@const editableGroups = [
                { label: $t("checkers.detail.admin-options"), opts: adminOpts },
                { label: $t("checkers.detail.configuration"), opts: userOpts },
            ]}
            {@const readOnlyGroups = [
                { key: "domainOpts", label: $t("checkers.option-groups.domain-settings"), opts: status.options?.domainOpts || [] },
                { key: "serviceOpts", label: $t("checkers.option-groups.service-settings"), opts: status.options?.serviceOpts || [] },
                { key: "runOpts", label: $t("checkers.option-groups.checker-parameters"), opts: status.options?.runOpts || [] },
            ]}
            {@const orphanedOpts = getOrphanedOptions(allEditableOpts, readOnlyGroups)}
            <Row class="mb-4">
                <Col md={6}>
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

                    {#if status.rules && status.rules.length > 0}
                        <CheckerRulesCard
                            rules={status.rules}
                            bind:optionValues
                            {inheritedValues}
                            {saving}
                            onsave={saveOptions}
                        />
                    {/if}
                </Col>

                <Col md={6}>
                    <CheckerOptionsPanel
                        {checkOptionsPromise}
                        {editableGroups}
                        {readOnlyGroups}
                        bind:optionValues
                        {inheritedValues}
                        {saving}
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
