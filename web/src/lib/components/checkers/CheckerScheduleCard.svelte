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
        FormGroup,
        Icon,
        Input,
        Label,
    } from "@sveltestrap/sveltestrap";
    import type { CheckerCheckIntervalSpec, HappydnsCheckPlan, HappydnsCheckPlanWritable } from "$lib/api-base/types.gen";
    import { t } from "$lib/translations";
    import { toasts } from "$lib/stores/toasts";
    import type { CheckerScope } from "$lib/api/checkers";
    import {
        getScopedCheckPlans,
        createScopedCheckPlan,
        updateScopedCheckPlan,
    } from "$lib/api/checkers";

    const NS_PER_MINUTE = 60_000_000_000;
    const NS_PER_HOUR = 3_600_000_000_000;

    interface Props {
        scope: CheckerScope;
        checkerId: string;
        plan: HappydnsCheckPlan | HappydnsCheckPlanWritable;
        intervalSpec?: CheckerCheckIntervalSpec;
    }

    let { scope, checkerId, plan = $bindable(), intervalSpec }: Props = $props();

    let existingPlanId = $state<string | undefined>(undefined);
    let saving = $state(false);

    // Determine whether to use minutes or hours as the UI unit.
    let useMinutes = $derived(
        intervalSpec != null && intervalSpec.min != null && intervalSpec.min < NS_PER_HOUR
    );
    let unitNs = $derived(useMinutes ? NS_PER_MINUTE : NS_PER_HOUR);

    let defaultIntervalNs = $derived(intervalSpec?.default ?? NS_PER_HOUR);
    let minNs = $derived(intervalSpec?.min ?? NS_PER_HOUR);
    let maxNs = $derived(intervalSpec?.max ?? 24 * NS_PER_HOUR);

    let schedulesPromise = $derived(getScopedCheckPlans(scope, checkerId));

    $effect(() => {
        schedulesPromise.then((schedules: HappydnsCheckPlan[]) => {
            if (schedules.length > 0) {
                const s = schedules[0];
                existingPlanId = s.id;
                plan = {
                    enabled: s.enabled ?? {},
                    interval: s.interval ?? defaultIntervalNs,
                };
            }
        });
    });

    async function save() {
        saving = true;
        try {
            const planData: HappydnsCheckPlanWritable = {
                enabled: plan.enabled,
                interval: plan.interval,
            };
            if (existingPlanId) {
                await updateScopedCheckPlan(scope, checkerId, existingPlanId, planData);
            } else {
                const created = await createScopedCheckPlan(scope, checkerId, planData);
                existingPlanId = created.id;
            }
            toasts.addToast({
                message: $t("checkers.schedule.saved"),
                type: "success",
                timeout: 5000,
            });
        } catch (error) {
            toasts.addErrorToast({
                message: $t("checkers.schedule.save-failed") + ": " + String(error),
                timeout: 10000,
            });
        } finally {
            saving = false;
        }
    }

    function intervalDisplayValue(): number {
        return Math.round((plan.interval ?? defaultIntervalNs) / unitNs);
    }

    function setIntervalValue(val: number) {
        const clamped = Math.max(minNs, Math.min(maxNs, val * unitNs));
        plan.interval = clamped;
    }

    const NS_PER_DAY = 24 * NS_PER_HOUR;
    const NS_PER_WEEK = 7 * NS_PER_DAY;

    function formatDuration(ns: number): string {
        if (ns >= NS_PER_WEEK && ns % NS_PER_WEEK === 0) {
            return `${ns / NS_PER_WEEK}w`;
        }
        if (ns >= NS_PER_DAY && ns % NS_PER_DAY === 0) {
            return `${ns / NS_PER_DAY}d`;
        }
        if (ns >= NS_PER_HOUR) {
            const h = Math.round(ns / NS_PER_HOUR);
            return `${h}h`;
        }
        const m = Math.round(ns / NS_PER_MINUTE);
        return `${m}min`;
    }
</script>

<Card class="mb-3">
    <CardHeader class="d-flex align-items-center justify-content-between">
        <strong>{$t("checkers.schedule.card-title")}</strong>
        <Button form="form-schedule" color="success" size="sm" onclick={save} disabled={saving}>
            {#if saving}
                <span class="spinner-border spinner-border-sm me-1"></span>
            {:else}
                <Icon name="check-circle"></Icon>
            {/if}
            {$t("checkers.schedule.save")}
        </Button>
    </CardHeader>
    <CardBody>
        <form id="form-schedule">
            <FormGroup>
                <Label>{$t("checkers.schedule.interval-label")}</Label>
                <div class="d-flex align-items-center gap-2">
                    <Input
                        type="number"
                        min={Math.round(minNs / unitNs)}
                        max={Math.round(maxNs / unitNs)}
                        value={intervalDisplayValue()}
                        oninput={(e: Event) =>
                            setIntervalValue(parseInt((e.target as HTMLInputElement).value) || 1)}
                        style="width: 100px"
                    />
                    <span>{useMinutes ? $t("checkers.schedule.minutes") : $t("checkers.schedule.hours")}</span>
                </div>
                <small class="text-muted">
                    {$t("checkers.schedule.interval-hint", {
                        intervalMin: formatDuration(minNs),
                        intervalMax: formatDuration(maxNs),
                        intervalDefault: formatDuration(defaultIntervalNs),
                    })}
                </small>
            </FormGroup>
        </form>

        {#if !existingPlanId}
            <Alert color="info" class="mb-0 mt-2">
                <Icon name="info-circle" />
                {$t("checkers.schedule.no-schedule-yet")}
            </Alert>
        {/if}
    </CardBody>
</Card>
