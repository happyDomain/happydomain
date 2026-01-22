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
    import { postTidy } from '$lib/api-admin';
    import { toasts } from '$lib/stores/toasts';

    let { class: className = "" } = $props();
    let isProcessing = $state(false);

    async function tidyDatabase() {
        if (!confirm("This will clean up orphaned records and optimize storage. Continue?"))
            return;

        isProcessing = true;

        try {
            await postTidy();

            toasts.addToast({
                type: "success",
                title: "Database tidied successfully!",
                timeout: 5000
            });
        } catch (err) {
            toasts.addErrorToast({
                message: err instanceof Error ? err.message : "Unknown error occurred"
            });
        } finally {
            isProcessing = false;
        }
    }
</script>

<section class={className}>
    <h2 class="h4 mb-3">Database Maintenance</h2>
    <div class="card">
        <div class="card-body">
            <p class="text-muted mb-3">
                Performs cleanup and maintenance operations on the database, removing orphaned records and optimizing storage.
            </p>
            <button
                type="button"
                class="btn btn-primary"
                disabled={isProcessing}
                onclick={tidyDatabase}
            >
                <i class="bi bi-arrow-repeat me-2"></i>
                {isProcessing ? "Processing..." : "Tidy Database"}
            </button>
        </div>
    </div>
</section>
