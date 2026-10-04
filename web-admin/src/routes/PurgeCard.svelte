<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2026 happyDomain
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
    import { postPurge } from "$lib/api-admin";
    import { toasts } from "$lib/stores/toasts";

    let { class: className = "" } = $props();
    let isProcessing = $state(false);

    let checkers = $state(true);
    let expiredSessions = $state(true);
    let notificationRecords = $state(true);
    let compact = $state(true);

    async function purgeDatabase() {
        if (!checkers && !expiredSessions && !notificationRecords) return;
        if (!confirm("The selected data will be deleted for good. Continue?")) return;

        isProcessing = true;

        try {
            const { data, error: err } = await postPurge({
                body: {
                    checkers,
                    expired_sessions: expiredSessions,
                    notification_records: notificationRecords,
                    compact,
                },
            });
            // On failure the body is still a report, holding what was
            // deleted before and despite the error.
            const report = (data ?? err) as any;
            if (err && typeof report?.error !== "string")
                throw new Error(report?.errmsg || String(err));

            const deleted = `Deleted ${report?.checkers ?? 0} checker keys, ${report?.expired_sessions ?? 0} expired sessions and ${report?.notification_records ?? 0} notification record keys.`;

            if (err) {
                toasts.addErrorToast({
                    title: "Purge partly failed",
                    message:
                        `${report.error} ${deleted}` +
                        (compact && !report.compacted ? " Storage not compacted." : ""),
                    timeout: 30000,
                });
                return;
            }

            toasts.addToast({
                type: "success",
                title: "Database purged",
                message:
                    deleted +
                    (compact
                        ? report?.compacted
                            ? " Storage compacted."
                            : " This storage backend cannot be compacted."
                        : ""),
                timeout: 10000,
            });
        } catch (err) {
            toasts.addErrorToast({
                message: err instanceof Error ? err.message : "Unknown error occurred",
                timeout: 10000,
            });
        } finally {
            isProcessing = false;
        }
    }
</script>

<div class="card border-danger h-100 {className}">
    <div class="card-body">
        <h3 class="h5 mb-3">Purge low value data</h3>
        <p class="text-muted mb-3">
            Frees space by deleting data the server recreates by itself or whose loss is minor.
        </p>
        <div class="form-check">
            <input
                class="form-check-input"
                type="checkbox"
                id="purge-checkers"
                bind:checked={checkers}
                disabled={isProcessing}
            />
            <label class="form-check-label" for="purge-checkers">
                Checker data: executions, evaluations, observations and their cache, discovery data
            </label>
        </div>
        <div class="form-check">
            <input
                class="form-check-input"
                type="checkbox"
                id="purge-sessions"
                bind:checked={expiredSessions}
                disabled={isProcessing}
            />
            <label class="form-check-label" for="purge-sessions">Expired sessions</label>
        </div>
        <div class="form-check mb-3">
            <input
                class="form-check-input"
                type="checkbox"
                id="purge-notifrecs"
                bind:checked={notificationRecords}
                disabled={isProcessing}
            />
            <label class="form-check-label" for="purge-notifrecs">
                History of sent notifications
            </label>
        </div>
        <button
            type="button"
            class="btn btn-danger"
            disabled={isProcessing || (!checkers && !expiredSessions && !notificationRecords)}
            onclick={purgeDatabase}
        >
            <i class="bi bi-trash me-2"></i>
            {isProcessing ? "Purging..." : "Purge Database"}
        </button>
    </div>
</div>
