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
     Col,
     Container,
     Icon,
     Row,
 } from 'sveltestrap';

 import ProviderSelector from '$lib/components/providers/Selector.svelte';
 import { t } from '$lib/translations';

 function selectNewProvider(event: CustomEvent<{ptype: string}>) {
     goto('new/' + encodeURIComponent(event.detail.ptype));
 }
</script>

<Container class="d-flex flex-column flex-fill" fluid>
    <h1 class="text-center my-2">
        <Button
            type="button"
            class="fw-bolder"
            color="link"
            on:click={() => history.go(-1)}
        >
            <Icon name="chevron-left" />
        </Button>
        {$t("provider.select-provider")}
    </h1>
    <hr class="mt-0 mb-0">
    <Row class="my-3 flex-grow-1">
        <Col md={{size: 8, offset: 2}}>
            <ProviderSelector on:provider-selected={selectNewProvider} />
        </Col>
    </Row>
</Container>
