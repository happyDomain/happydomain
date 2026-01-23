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
        ListGroup,
        ListGroupItem,
    } from "@sveltestrap/sveltestrap";

    interface Props {
        domainId: string;
        uid: string;
        zoneHistory: string[];
    }

    let { domainId = "", uid = "", zoneHistory = [] }: Props = $props();
</script>

<Card>
    <CardHeader>
        <div class="d-flex justify-content-between align-items-center">
            <h5 class="mb-0">
                <Icon name="clock-history"></Icon>
                Zone History
            </h5>
            <Badge color="secondary">{zoneHistory.length} zones</Badge>
        </div>
    </CardHeader>
    {#if zoneHistory.length === 0}
        <CardBody>
            <p class="text-muted mb-0">No zone history available.</p>
        </CardBody>
    {:else}
        <ListGroup flush>
            {#each zoneHistory as zoneId, index}
                <ListGroupItem href="/users/{uid}/domains/{domainId}/zones/{zoneId}" action>
                    <Badge color="info" class="me-2">#{zoneHistory.length - index}</Badge>
                    <code>{zoneId}</code>
                </ListGroupItem>
            {/each}
        </ListGroup>
    {/if}
</Card>
