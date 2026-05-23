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
    import { Alert, Button, Spinner } from "@sveltestrap/sveltestrap";

    import { exportUserData } from "$lib/api/user";
    import { userSession } from "$lib/stores/usersession";
    import { t } from "$lib/translations";
    import { downloadBlob } from "$lib/utils/checkers";

    let loading = false;
    let error: string | null = null;
    let partialErrors: string[] = [];

    async function downloadUserData() {
        loading = true;
        error = null;
        partialErrors = [];
        try {
            const { text, errors } = await exportUserData($userSession);
            partialErrors = errors;
            downloadBlob(text, "happydomain-export.json", "application/json");
        } catch (e) {
            error = e instanceof Error ? e.message : String(e);
        } finally {
            loading = false;
        }
    }
</script>

<Alert color="warning" class="d-flex align-items-start gap-2">
    <i class="bi bi-exclamation-triangle-fill flex-shrink-0 mt-1"></i>
    {$t("account.export.warning")}
</Alert>
<Button color="outline-primary" on:click={downloadUserData} disabled={loading}>
    {#if loading}
        <Spinner size="sm" />
    {/if}
    {$t("account.export.button")}
</Button>

{#if error}
    <Alert color="danger" class="mt-2 d-flex align-items-start gap-2">
        <i class="bi bi-x-circle-fill flex-shrink-0 mt-1"></i>
        {error}
    </Alert>
{/if}
{#if partialErrors.length > 0}
    <Alert color="warning" class="mt-2">
        <div class="d-flex align-items-start gap-2">
            <i class="bi bi-exclamation-triangle-fill flex-shrink-0 mt-1"></i>
            {$t("account.export.partial")}
        </div>
        <ul class="mb-0 mt-1 ms-4">
            {#each partialErrors as e}
                <li>{e}</li>
            {/each}
        </ul>
    </Alert>
{/if}
