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
    import { Alert, Button, Container, Icon, Spinner } from "@sveltestrap/sveltestrap";

    import { getUsersByUidDomainsByDomainZones } from "$lib/api-admin";
    import ZoneHistoryCard from "./ZoneHistoryCard.svelte";

    const uid = $derived(page.params.uid ?? "");
    const domainId = $derived(page.params.domain ?? "");
    let zonesQ = $derived(getUsersByUidDomainsByDomainZones({ path: { uid, domain: domainId } }));

    let zoneHistory = $state<string[]>([]);

    // Load zones data when promise resolves
    $effect(() => {
        zonesQ.then((response) => {
            if (response?.data) {
                // Flatten the 2D array and convert numbers to strings
                zoneHistory = response.data.flat().map((id) => String(id));
            }
        });
    });
</script>

<Container class="flex-fill my-5">
    <div class="d-flex align-items-center gap-1 mb-4">
        <Button color="link" href="/users/{uid}/domains/{domainId}" class="text-black">
            <Icon name="chevron-left"></Icon>
        </Button>
        <h1 class="display-5 mb-0">Zone History</h1>
    </div>

    {#await zonesQ}
        <div class="text-center my-5">
            <Spinner color="primary" />
            <p class="mt-3">Loading zones...</p>
        </div>
    {:then zonesR}
        {#if zonesR?.data}
            <ZoneHistoryCard {domainId} {uid} {zoneHistory} />
        {:else}
            <Alert color="warning">
                <h4 class="alert-heading">No data available</h4>
                <p>The zones response did not contain any data.</p>
                <hr />
                <Button
                    type="button"
                    color="secondary"
                    outline
                    href="/users/{uid}/domains/{domainId}"
                >
                    <Icon name="arrow-left"></Icon>
                    Back to Domain
                </Button>
            </Alert>
        {/if}
    {:catch error}
        <Alert color="danger">
            <h4 class="alert-heading">Error loading zones</h4>
            <p>{error}</p>
            <hr />
            <Button type="button" color="secondary" outline href="/users/{uid}/domains/{domainId}">
                <Icon name="arrow-left"></Icon>
                Back to Domain
            </Button>
        </Alert>
    {/await}
</Container>
