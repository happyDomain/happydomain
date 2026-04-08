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
        Form,
        FormGroup,
        FormText,
        Icon,
        Input,
        Label,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { putUsersByUid } from "$lib/api-admin";
    import { toasts } from "$lib/stores/toasts";

    import type { HappydnsUser, HappydnsUserQuota } from "$lib/api-admin";

    interface UserQuotaCardProps {
        user: HappydnsUser;
        uid: string;
    }

    let { user, uid }: UserQuotaCardProps = $props();

    let maxChecksPerDay = $state(0);
    let retentionDays = $state(0);
    let inactivityPauseDays = $state(0);
    let schedulingPaused = $state(false);
    let updatedAt = $state<string | undefined>(undefined);

    let loading = $state(false);
    let errorMessage = $state("");

    $effect(() => {
        const q: HappydnsUserQuota = user?.quota ?? {};
        maxChecksPerDay = q.max_checks_per_day ?? 0;
        retentionDays = q.retention_days ?? 0;
        inactivityPauseDays = q.inactivity_pause_days ?? 0;
        schedulingPaused = q.scheduling_paused ?? false;
        updatedAt = q.updated_at;
    });

    async function handleSubmit(e: Event) {
        e.preventDefault();
        loading = true;
        errorMessage = "";

        try {
            const body: any = {
                email: user.email,
                created_at: user.created_at,
                last_seen: user.last_seen,
                settings: user.settings,
                quota: {
                    max_checks_per_day: Number(maxChecksPerDay) || 0,
                    retention_days: Number(retentionDays) || 0,
                    inactivity_pause_days: Number(inactivityPauseDays) || 0,
                    scheduling_paused: schedulingPaused,
                },
            };

            const res = await putUsersByUid({ path: { uid }, body });
            const updated = (res?.data as HappydnsUser | undefined)?.quota;
            if (updated?.updated_at) updatedAt = updated.updated_at;

            toasts.addToast({
                message: "Quota updated successfully",
                type: "success",
                timeout: 5000,
            });
        } catch (error) {
            errorMessage = "Failed to update quota: " + error;
            toasts.addErrorToast({ message: errorMessage, timeout: 10000 });
        } finally {
            loading = false;
        }
    }
</script>

<Card class="mb-4">
    <CardHeader class="d-flex justify-content-between align-items-center">
        <h5 class="mb-0">
            <Icon name="speedometer2" class="me-2"></Icon>
            Admin Quota
        </h5>
        {#if updatedAt}
            <small class="text-muted">
                Updated {new Date(updatedAt).toLocaleString()}
            </small>
        {/if}
    </CardHeader>
    <CardBody>
        <p class="text-muted small">
            These limits are controlled by administrators and cannot be modified
            by the user. A value of <code>0</code> means "use the system default".
        </p>

        {#if errorMessage}
            <Alert color="danger" dismissible fade>{errorMessage}</Alert>
        {/if}

        <Form onsubmit={handleSubmit}>
            <FormGroup>
                <Label for="schedulingPaused" class="form-check-label">
                    <Input
                        type="checkbox"
                        id="schedulingPaused"
                        bind:checked={schedulingPaused}
                    />
                    Pause scheduler for this user
                </Label>
                <FormText>
                    Admin kill switch — when enabled, no checks will run for this
                    user regardless of their plans.
                </FormText>
            </FormGroup>

            <FormGroup>
                <Label for="retentionDays">Retention (days)</Label>
                <Input
                    type="number"
                    id="retentionDays"
                    min="0"
                    bind:value={retentionDays}
                />
                <FormText>
                    Maximum age of stored check executions. Older entries are
                    pruned by the janitor according to the tiered retention policy.
                </FormText>
            </FormGroup>

            <FormGroup>
                <Label for="maxChecksPerDay">Max checks per day</Label>
                <Input
                    type="number"
                    id="maxChecksPerDay"
                    min="0"
                    bind:value={maxChecksPerDay}
                />
                <FormText>
                    Daily cap on the number of executions the scheduler may launch
                    for this user (enforced later).
                </FormText>
            </FormGroup>

            <FormGroup>
                <Label for="inactivityPauseDays">
                    Inactivity pause (days)
                </Label>
                <Input
                    type="number"
                    id="inactivityPauseDays"
                    bind:value={inactivityPauseDays}
                />
                <FormText>
                    The scheduler stops running checks after this many days
                    without login. Use a negative value to disable.
                </FormText>
            </FormGroup>

            <Button color="primary" type="submit" disabled={loading}>
                {#if loading}
                    <Spinner size="sm" class="me-2" />
                {:else}
                    <Icon name="check-circle" class="me-2"></Icon>
                {/if}
                Save Quota
            </Button>
        </Form>
    </CardBody>
</Card>
