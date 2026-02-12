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
    import { page } from "$app/state";
    import { Button, Card, Icon, Spinner } from "@sveltestrap/sveltestrap";

    import CheckerOptionsCard from "$lib/components/checkers/CheckerOptionsCard.svelte";
    import CheckerScheduleCard from "$lib/components/checkers/CheckerScheduleCard.svelte";
    import PageTitle from "$lib/components/PageTitle.svelte";
    import { t } from "$lib/translations";
    import {
        listAvailableCheckers,
        updateCheckSchedule,
        createCheckSchedule,
        getCheckStatus,
        getDomainCheckOptions,
        updateDomainCheckOptions,
    } from "$lib/api/checkers";
    import type { Domain } from "$lib/model/domain";
    import { CheckScopeType, type AvailableChecker, type CheckerInfo } from "$lib/model/checker";
    import { checkers } from "$lib/stores/checkers";
    import { toasts } from "$lib/stores/toasts";

    interface Props {
        data: { domain: Domain };
    }

    let { data }: Props = $props();

    const checkName = $derived(page.params.cname || "");
    const checkDisplayName = $derived($checkers?.[checkName]?.name || checkName);

    // Resolved check data
    let check = $state<AvailableChecker | null>(null);
    let checkStatus = $state<CheckerInfo | null>(null);
    let loading = $state(true);
    let loadError = $state<string | null>(null);

    // Form state
    let formEnabled = $state(true);
    let formIntervalHours = $state(24);
    let saving = $state(false);

    // Options state
    let domainOptionValues = $state<Record<string, any>>({});

    async function loadCheck() {
        loading = true;
        loadError = null;
        try {
            const [checks, status, options] = await Promise.all([
                listAvailableCheckers(data.domain.id),
                getCheckStatus(checkName),
                getDomainCheckOptions(data.domain.id, checkName),
            ]);
            const found = checks?.find((c) => c.checker_name === checkName) ?? null;
            check = found;
            checkStatus = status;
            domainOptionValues = { ...(options || {}) };
            if (found) {
                formEnabled = found.enabled;
                formIntervalHours =
                    found.schedule &&
                    found.schedule.interval !== undefined &&
                    found.schedule.interval > 0
                        ? found.schedule.interval / (3600 * 1e9)
                        : 24;
            }
        } catch (e: any) {
            loadError = e.message;
        } finally {
            loading = false;
        }
    }

    loadCheck();

    async function handleSave() {
        if (!check) return;
        saving = true;

        try {
            const intervalNs = Math.max(formIntervalHours, 1) * 3600 * 1e9;

            if (check.schedule) {
                await updateCheckSchedule(check.schedule.id!, {
                    ...check.schedule,
                    enabled: formEnabled,
                    interval: intervalNs,
                });
            } else {
                await createCheckSchedule({
                    checker_name: check.checker_name,
                    target_type: CheckScopeType.CheckScopeDomain,
                    target_id: data.domain.id,
                    interval: intervalNs,
                    enabled: formEnabled,
                });
            }

            toasts.addToast({
                title: $t("checkers.schedule.saved"),
                type: "success",
                timeout: 3000,
            });
            await loadCheck();
        } catch (e: any) {
            toasts.addErrorToast({
                title: $t("checkers.schedule.save-failed"),
                message: e.message,
            });
        } finally {
            saving = false;
        }
    }
</script>

<svelte:head>
    <title>
        {checkName} - {$t("checkers.schedule.title")} - {data.domain.domain} - happyDomain
    </title>
</svelte:head>

<div class="flex-fill pb-4 pt-2">
    <PageTitle
        title={$t("checkers.schedule.title")}
        domain={data.domain.domain}
        subtitle={checkDisplayName}
    >
        <Button
            color="info"
            href={`/domains/${encodeURIComponent(data.domain.domain)}/checks/${encodeURIComponent(checkName)}/results`}
        >
            <Icon name="bar-chart-fill"></Icon>
            {$t("checkers.list.view-results")}
        </Button>
    </PageTitle>

    {#if loading}
        <div class="mt-5 text-center flex-fill">
            <Spinner />
            <p>{$t("checkers.list.loading")}</p>
        </div>
    {:else if loadError}
        <Card body color="danger">
            <p class="mb-0">
                <Icon name="exclamation-triangle-fill"></Icon>
                {$t("checkers.list.error-loading", { error: loadError })}
            </p>
        </Card>
    {:else if !check}
        <Card body>
            <p class="text-center text-muted mb-0">
                <Icon name="info-circle"></Icon>
                {$t("checkers.list.no-checks")}
            </p>
        </Card>
    {:else}
        <CheckerScheduleCard
            checker={check}
            bind:formEnabled
            bind:formIntervalHours
            {saving}
            onSave={handleSave}
        />

        <CheckerOptionsCard
            options={checkStatus?.options?.domainOpts ?? []}
            bind:optionValues={domainOptionValues}
            title={$t("checkers.option-groups.domain-settings")}
            saveOptionsFn={(values) => updateDomainCheckOptions(data.domain.id, checkName, values)}
        />
    {/if}
</div>
