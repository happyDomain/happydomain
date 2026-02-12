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
    import { Alert, Button, Col, Container, Icon, Row, Spinner } from "@sveltestrap/sveltestrap";

    import { getUsersByUidDomainsByDomain } from "$lib/api-admin";
    import DomainInformationCard from "./DomainInformationCard.svelte";
    import ZoneHistoryCard from "./zones/ZoneHistoryCard.svelte";

    const uid = $derived(page.params.uid!);
    const domainId = $derived(page.params.domain!);
    let domainQ = $derived(getUsersByUidDomainsByDomain({ path: { uid, domain: domainId } }));

    let zoneHistory = $state<string[]>([]);

    // Load domain data when promise resolves
    $effect(() => {
        domainQ.then((response) => {
            if (response?.data && response.data.length > 0) {
                zoneHistory = response.data[0].zone_history || [];
            }
        });
    });
</script>

<Container class="flex-fill my-5">
    <Row class="mb-4">
        <Col>
            <h1 class="display-5">
                <Icon name="pencil"></Icon>
                Edit Domain
            </h1>
        </Col>
    </Row>

    {#await domainQ}
        <div class="text-center my-5">
            <Spinner color="primary" />
            <p class="mt-3">Loading domain...</p>
        </div>
    {:then domainR}
        {#if domainR?.data && domainR.data.length > 0}
            {@const domain = domainR.data[0]}
            <Row>
                <Col md={8} lg={6}>
                    <DomainInformationCard domainData={domain} {uid} {domainId} />
                </Col>

                <Col md={8} lg={6}>
                    <ZoneHistoryCard {domainId} {uid} {zoneHistory} />
                </Col>
            </Row>
        {:else}
            <Alert color="warning">
                <h4 class="alert-heading">No data available</h4>
                <p>The domain response did not contain any data.</p>
                <hr />
                <Button type="button" color="secondary" outline href="/domains">
                    <Icon name="arrow-left"></Icon>
                    Back to Domains
                </Button>
            </Alert>
        {/if}
    {:catch error}
        <Alert color="danger">
            <h4 class="alert-heading">Error loading domain</h4>
            <p>{error}</p>
            <hr />
            <Button type="button" color="secondary" outline href="/domains">
                <Icon name="arrow-left"></Icon>
                Back to Domains
            </Button>
        </Alert>
    {/await}
</Container>
