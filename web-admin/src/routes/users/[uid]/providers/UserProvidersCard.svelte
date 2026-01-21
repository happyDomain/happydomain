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
        Badge,
        Card,
        CardBody,
        CardHeader,
        Icon,
        ListGroup,
        ListGroupItem,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import type { HappydnsProviderMeta, HappydnsErrorResponse } from '$lib/api-admin';

    interface UserProvidersCardProps {
        providersQ: Promise<(
            | { data: HappydnsProviderMeta[]; error: undefined }
            | { data: undefined; error: HappydnsErrorResponse }
        ) & { request: Request; response: Response }>;
        userId: string;
    }

    let { providersQ, userId }: UserProvidersCardProps = $props();
</script>

{#await providersQ}
    <Card>
        <CardBody>
            <div class="text-center">
                <Spinner color="primary" size="sm" />
                <span class="ms-2">Loading providers...</span>
            </div>
        </CardBody>
    </Card>
{:then providersR}
    {@const userProviders = providersR.data || []}
    <Card>
        <CardHeader>
            <div class="d-flex justify-content-between align-items-center">
                <h5 class="mb-0">
                    <Icon name="cloud"></Icon>
                    User Providers
                </h5>
                <Badge color="secondary">{userProviders.length} providers</Badge>
            </div>
        </CardHeader>
        {#if userProviders.length === 0}
            <CardBody>
                <p class="text-muted mb-0">This user has no providers.</p>
            </CardBody>
        {:else}
            <ListGroup flush>
                {#each userProviders as provider}
                    <ListGroupItem href="/users/{userId}/providers/{provider._id}" action>
                        <div class="d-flex justify-content-between align-items-start">
                            <div>
                                <strong>{provider._comment || 'Unnamed provider'}</strong>
                                {#if provider._srctype}
                                    <Badge color="info" class="ms-2">{provider._srctype}</Badge>
                                {/if}
                                <div class="small text-muted">
                                    <code>{provider._id}</code>
                                </div>
                            </div>
                        </div>
                    </ListGroupItem>
                {/each}
            </ListGroup>
        {/if}
    </Card>
{:catch}
    <Card>
        <CardBody>
            <Alert color="warning" class="mb-0">
                Unable to load providers.
            </Alert>
        </CardBody>
    </Card>
{/await}
