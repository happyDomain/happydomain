<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2024 happyDomain
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

 import {
     Button,
     Container,
     Col,
     Icon,
     Row,
     Spinner,
 } from '@sveltestrap/sveltestrap';

 import ProviderList from '$lib/components/providers/List.svelte';
 import { providers, refreshProviders } from '$lib/stores/providers';
 import { t } from '$lib/translations';

 refreshProviders();
</script>

<Container class="flex-fill pt-4 pb-5">
    {#if !window.disable_providers}
        <Button
            type="button"
            color="primary"
            class="float-end"
            on:click={() => goto('providers/new')}
        >
            <Icon name="plus" />
            {$t('common.add-new-thing', { thing: $t('provider.kind') })}
        </Button>
    {/if}
    <h1 class="text-center mb-4">
        {$t('provider.title')}
    </h1>
    {#if !$providers}
        <div class="d-flex justify-content-center">
            <Spinner color="primary" />
        </div>
    {:else}
        <Row>
            <Col md={{size: 8, offset: 2}}>
                <ProviderList
                    items={$providers}
                    on:new-provider={() => goto('providers/new')}
                    on:click={(event) => goto('providers/' + encodeURIComponent(event.detail._id))}
                />
            </Col>
        </Row>
    {/if}
</Container>
