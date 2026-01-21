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
    import { putBackupJson } from '$lib/api-admin';

    let file: File | null = $state(null);

    let { class: className = "" } = $props();

    function handleFileChange(event: Event) {
        const target = event.target as HTMLInputElement;
        if (target.files && target.files.length > 0) {
            file = target.files[0];
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
            } catch (e) {
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
                    <form action="/api/backup.json" method="post">
                        <button class="btn btn-primary w-100">
                            <i class="bi bi-download me-2"></i>
                            Download Database Backup
                        </button>
                    </form>
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
