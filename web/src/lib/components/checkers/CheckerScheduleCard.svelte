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
        CardBody,
        CardHeader,
        Icon,
        Input,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import type { AvailableChecker } from "$lib/model/checker";
    import { formatCheckDate, formatRelative } from "$lib/utils";

    interface Props {
        checker: AvailableChecker;
        formEnabled: boolean;
        formIntervalHours: number;
        saving: boolean;
        onSave: () => void;
    }

    let { checker, formEnabled = $bindable(), formIntervalHours = $bindable(), saving, onSave }: Props = $props();
</script>

<Card class="mb-4">
    <CardHeader>
        <h4 class="mb-0">
            <Icon name="clock-history"></Icon>
            {$t("checkers.schedule.card-title")}
        </h4>
    </CardHeader>
    <CardBody>
        <div class="mb-4">
            <div class="form-check form-switch">
                <input
                    class="form-check-input"
                    type="checkbox"
                    role="switch"
                    id="schedule-enabled"
                    bind:checked={formEnabled}
                    disabled={saving}
                />
                <label class="form-check-label" for="schedule-enabled">
                    {#if formEnabled}
                        <Badge color="success">{$t("checkers.schedule.auto-enabled")}</Badge>
                    {:else}
                        <Badge color="secondary">{$t("checkers.schedule.auto-disabled")}</Badge>
                    {/if}
                </label>
            </div>
        </div>

        {#if formEnabled}
            <div class="mb-4">
                <label for="schedule-interval" class="form-label fw-semibold">
                    {$t("checkers.schedule.interval-label")}
                </label>
                <div class="input-group" style="max-width: 300px;">
                    <Input
                        type="number"
                        id="schedule-interval"
                        min={1}
                        step={1}
                        bind:value={formIntervalHours}
                        disabled={saving}
                    />
                    <span class="input-group-text">
                        {$t("checkers.schedule.hours")}
                    </span>
                </div>
                <div class="form-text">
                    {$t("checkers.schedule.interval-hint")}
                </div>
            </div>
        {/if}

        {#if checker.schedule}
            <div class="mb-4">
                <div class="row g-3">
                    {#if checker.schedule.last_run}
                        <div class="col-auto">
                            <span class="text-muted fw-semibold">
                                {$t("checkers.schedule.last-run")}:
                            </span>
                            <span>
                                {formatCheckDate(checker.schedule.last_run, "medium", $t)}
                                <small class="text-muted">
                                    ({formatRelative(checker.schedule.last_run, $t)})
                                </small>
                            </span>
                        </div>
                    {/if}
                    {#if checker.enabled && checker.schedule.next_run}
                        <div class="col-auto">
                            <span class="text-muted fw-semibold">
                                {$t("checkers.schedule.next-run")}:
                            </span>
                            <span>
                                {formatCheckDate(checker.schedule.next_run, "medium", $t)}
                                <small class="text-muted">
                                    ({formatRelative(checker.schedule.next_run, $t)})
                                </small>
                            </span>
                        </div>
                    {/if}
                </div>
            </div>
        {:else}
            <p class="text-muted">
                <Icon name="info-circle"></Icon>
                {$t("checkers.schedule.no-schedule-yet")}
            </p>
        {/if}

        <Button color="primary" disabled={saving} onclick={onSave}>
            {#if saving}
                <Spinner size="sm" class="me-1" />
            {/if}
            <Icon name="check-lg"></Icon>
            {$t("checkers.schedule.save")}
        </Button>
    </CardBody>
</Card>
