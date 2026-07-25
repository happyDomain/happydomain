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
    import { goto } from '$app/navigation';
    import { authLinks } from '$links';

    import { putBackupJson } from '$lib/api-admin';
    import { clearAdminToken, getAdminToken } from '$lib/stores/adminsession';
    import { downloadBlob } from '$lib/utils/checkers';

    let file: File | null = $state(null);

    let { class: className = "" } = $props();

    function handleFileChange(event: Event) {
        const target = event.target as HTMLInputElement;
        if (target.files && target.files.length > 0) {
            file = target.files[0];
        }
    }

    // downloadBackup fetches the full backup through a script-driven request so
    // it carries the admin bearer token (a plain <form> POST would not), then
    // triggers a client-side file download. It cannot go through the generated
    // api-admin client, which parses responses as JSON, so it mirrors that
    // client's expired-session handling explicitly.
    async function downloadBackup() {
        try {
            const headers: Record<string, string> = {};
            const token = getAdminToken();
            if (token) headers["Authorization"] = "Bearer " + token;

            const response = await fetch("/api/backup.json", { method: "POST", headers });

            if (response.status === 401) {
                clearAdminToken();
                // The login path is already resolved; the one handed over to it
                // is read from the address bar, base path included, so the
                // login page can send the user back to it as is.
                // eslint-disable-next-line svelte/no-navigation-without-resolve
                goto(authLinks().login() + "?next=" + encodeURIComponent(window.location.pathname));
                return;
            }

            if (!response.ok) {
                alert("Backup failed!");
                return;
            }

            downloadBlob(await response.blob(), "happydomain-backup.json", "application/json");
        } catch (err) {
            console.error("Error:", err);
            alert("Backup failed!");
        }
    }

    async function restoreBackup() {
        if (!confirm("Warning: This will overwrite the existing data. Continue?"))
            return;

        if (!file) return;

        try {
            const text = await file.text();

            let datajson;
            try {
                datajson = JSON.parse(text);
            } catch {
                alert("The file is not valid JSON!");
                return;
            }

            const response = await putBackupJson({ body: datajson });

            console.log("Restore successful:", response);
            alert("Database restored successfully!");
        } catch (err) {
            console.error("Error:", err);
            alert("Restore failed!");
        }
    }
</script>

<section class={className}>
    <h2 class="h4 mb-3">Database Management</h2>
    <div class="card">
        <div class="card-body">
            <div class="row g-3">
                <div class="col-md-6">
                    <button type="button" class="btn btn-primary w-100" onclick={downloadBackup}>
                        <i class="bi bi-download me-2"></i>
                        Download Database Backup
                    </button>
                </div>
                <div class="col-md-6">
                    <div class="input-group">
                        <input
                            type="file"
                            class="form-control"
                            accept=".json"
                            onchange={handleFileChange}
                        />
                        <button
                            type="button"
                            class="btn btn-primary"
                            disabled={file == null}
                            onclick={restoreBackup}
                        >
                            <i class="bi bi-upload me-2"></i>
                            Restore
                        </button>
                    </div>
                </div>
            </div>
        </div>
    </div>
</section>
