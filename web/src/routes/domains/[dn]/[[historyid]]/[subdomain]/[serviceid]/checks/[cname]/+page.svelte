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
        listServiceAvailableCheckers,
        updateCheckSchedule,
        createCheckSchedule,
        getCheckStatus,
        getServiceCheckOptions,
        updateServiceCheckOptions,
    } from "$lib/api/checkers";
    import type { Domain } from "$lib/model/domain";
    import { CheckScopeType, type AvailableChecker, type CheckerInfo } from "$lib/model/checker";
    import { checkers } from "$lib/stores/checkers";
    import { toasts } from "$lib/stores/toasts";

    interface Props {
        data: { domain: Domain; zoneId: string; subdomain: string; serviceid: string };
    }

    let { data }: Props = $props();

    const NS_PER_HOUR = 3600 * 1e9;

    const checkerName = $derived(page.params.cname || "");
    const checkerDisplayName = $derived($checkers?.[checkerName]?.name || checkerName);
    const intervalSpec = $derived($checkers?.[checkerName]?.interval);

    function serviceChecksBasePath(): string {
        const dn = encodeURIComponent(data.domain.domain);
        const historyid = page.params.historyid ? encodeURIComponent(page.params.historyid) : "";
        const sub = encodeURIComponent(page.params.subdomain!);
        const svc = encodeURIComponent(data.serviceid);
        return `/domains/${dn}/${historyid}/${sub}/${svc}/checks`;
    }

    // Resolved check data
    let checker = $state<AvailableChecker | null>(null);
    let checkStatus = $state<CheckerInfo | null>(null);
    let loading = $state(true);
    let loadError = $state<string | null>(null);

    // Form state
    let formEnabled = $state(true);
    let formIntervalHours = $state(24);
    let saving = $state(false);

    // Options state
    let serviceOptionValues = $state<Record<string, any>>({});

    async function loadChecker() {
        loading = true;
        loadError = null;
        try {
            const [checkerList, status, options] = await Promise.all([
                listServiceAvailableCheckers(
                    data.domain.id,
                    data.zoneId,
                    data.subdomain,
                    data.serviceid,
                ),
                getCheckStatus(checkerName),
                getServiceCheckOptions(
                    data.domain.id,
                    data.zoneId,
                    data.subdomain,
                    data.serviceid,
                    checkerName,
                ),
            ]);
            const found = checkerList?.find((c) => c.checker_name === checkerName) ?? null;
            checker = found;
            checkStatus = status;
            serviceOptionValues = { ...(options || {}) };
            if (found) {
                formEnabled = found.enabled;
                const defaultHours = intervalSpec?.default ? intervalSpec.default / NS_PER_HOUR : 24;
                formIntervalHours =
                    found.schedule &&
                    found.schedule.interval !== undefined &&
                    found.schedule.interval > 0
                        ? found.schedule.interval / NS_PER_HOUR
                        : defaultHours;
            }
        } catch (e: any) {
            loadError = e.message;
        } finally {
            loading = false;
        }
    }

    loadChecker();

    async function handleSave() {
        if (!checker) return;
        saving = true;

        try {
            const intervalNs = Math.max(formIntervalHours, 1) * 3600 * 1e9;

            if (checker.schedule) {
                await updateCheckSchedule(checker.schedule.id!, {
                    ...checker.schedule,
                    enabled: formEnabled,
                    interval: intervalNs,
                });
            } else {
                await createCheckSchedule({
                    checker_name: checker.checker_name,
                    target_type: CheckScopeType.CheckScopeService,
                    target_id: data.serviceid,
                    interval: intervalNs,
                    enabled: formEnabled,
                });
            }

            toasts.addToast({
                title: $t("checkers.schedule.saved"),
                type: "success",
                timeout: 3000,
            });
            await loadChecker();
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
        {checkerName} - {$t("checkers.schedule.title")} - {data.domain.domain} - happyDomain
    </title>
</svelte:head>

<div class="flex-fill pb-4 pt-2">
    <PageTitle
        title={$t("checkers.schedule.title")}
        domain={data.domain.domain}
        subtitle={checkerDisplayName}
    >
        <Button
            color="info"
            href={`${serviceChecksBasePath()}/${encodeURIComponent(checkerName)}/results`}
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
    {:else if !checker}
        <Card body>
            <p class="text-center text-muted mb-0">
                <Icon name="info-circle"></Icon>
                {$t("checkers.list.no-checks-service")}
            </p>
        </Card>
    {:else}
        <CheckerScheduleCard
            {checker}
            {intervalSpec}
            bind:formEnabled
            bind:formIntervalHours
            {saving}
            onSave={handleSave}
        />

        <CheckerOptionsCard
            options={checkStatus?.options?.serviceOpts ?? []}
            bind:optionValues={serviceOptionValues}
            title={$t("checkers.option-groups.service-settings")}
            saveOptionsFn={(values) =>
                updateServiceCheckOptions(
                    data.domain.id,
                    data.zoneId,
                    data.subdomain,
                    data.serviceid,
                    checkerName,
                    values,
                )}
        />
    {/if}
</div>
