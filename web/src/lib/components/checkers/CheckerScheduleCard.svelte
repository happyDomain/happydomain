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
    import { Card, CardBody, CardHeader, Icon, Input, Label } from "@sveltestrap/sveltestrap";
    import type {
        CheckerCheckIntervalSpec,
        HappydnsCheckPlan,
        HappydnsCheckPlanWritable,
    } from "$lib/api-base/types.gen";
    import { t } from "$lib/translations";
    import { toasts } from "$lib/stores/toasts";
    import { appConfig } from "$lib/stores/config";
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
    let saveStatus = $state<"idle" | "saving" | "saved" | "error">("idle");
    let debounceTimer: ReturnType<typeof setTimeout> | undefined;

    let useMinutes = $derived(
        intervalSpec != null && intervalSpec.min != null && intervalSpec.min < NS_PER_HOUR,
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
                    disabled: s.disabled ?? false,
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
                disabled: plan.disabled ?? false,
            };
            if (existingPlanId) {
                await updateScopedCheckPlan(scope, checkerId, existingPlanId, planData);
            } else {
                const created = await createScopedCheckPlan(scope, checkerId, planData);
                existingPlanId = created.id;
            }
        } finally {
            saving = false;
        }
    }

    export { save };

    function triggerAutosave() {
        clearTimeout(debounceTimer);
        saveStatus = "idle";
        debounceTimer = setTimeout(async () => {
            saveStatus = "saving";
            try {
                await save();
                saveStatus = "saved";
                setTimeout(() => {
                    if (saveStatus === "saved") saveStatus = "idle";
                }, 2000);
            } catch (error) {
                saveStatus = "error";
                toasts.addErrorToast({
                    message: $t("checkers.schedule.save-failed") + ": " + String(error),
                    timeout: 10000,
                });
            }
        }, 700);
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

    let isEnabled = $derived(!$appConfig.disable_checker_scheduler && !(plan.disabled ?? false));
</script>

<Card class="mb-3">
    <CardHeader class="d-flex align-items-center gap-3">
        <strong class="me-auto">{$t("checkers.schedule.card-title")}</strong>

        {#if saveStatus === "saving"}
            <span class="spinner-border spinner-border-sm text-muted"></span>
        {:else if saveStatus === "saved"}
            <span class="text-success small d-flex align-items-center gap-1">
                <Icon name="check-circle" />
                {$t("checkers.schedule.saved")}
            </span>
        {:else if saveStatus === "error"}
            <span class="text-danger small d-flex align-items-center gap-1">
                <Icon name="exclamation-circle" />
                {$t("checkers.schedule.save-failed")}
            </span>
        {/if}

        <div class="d-flex align-items-center gap-2">
            <Label for="schedule-enabled-toggle" class="mb-0 text-nowrap user-select-none">
                <span class={isEnabled ? "text-success fw-semibold" : "text-muted"}>
                    {isEnabled ? $t("checkers.schedule.enabled") : $t("checkers.schedule.disabled")}
                </span>
            </Label>
            <Input
                type="switch"
                id="schedule-enabled-toggle"
                disabled={$appConfig.disable_checker_scheduler}
                checked={isEnabled}
                onchange={(e: Event) => {
                    plan.disabled = !(e.target as HTMLInputElement).checked;
                    triggerAutosave();
                }}
            />
        </div>
    </CardHeader>

    <CardBody>
        {#if isEnabled}
            <div>
                <Label class="mb-1">{$t("checkers.schedule.interval-label")}</Label>
                <div class="d-flex align-items-center gap-2">
                    <Input
                        type="number"
                        min={Math.round(minNs / unitNs)}
                        max={Math.round(maxNs / unitNs)}
                        value={intervalDisplayValue()}
                        oninput={(e: Event) => {
                            setIntervalValue(parseInt((e.target as HTMLInputElement).value) || 1);
                            triggerAutosave();
                        }}
                        style="width: 100px"
                    />
                    <span
                        >{useMinutes
                            ? $t("checkers.schedule.minutes")
                            : $t("checkers.schedule.hours")}</span
                    >
                </div>
                <small class="text-muted mt-1 d-block">
                    {$t("checkers.schedule.interval-hint", {
                        intervalMin: formatDuration(minNs),
                        intervalMax: formatDuration(maxNs),
                        intervalDefault: formatDuration(defaultIntervalNs),
                    })}
                </small>
            </div>
        {:else}
            <p class="text-muted mb-0 d-flex align-items-center gap-2">
                <Icon name="pause-circle" />
                {$t("checkers.schedule.paused-hint")}
            </p>
        {/if}
    </CardBody>
</Card>
