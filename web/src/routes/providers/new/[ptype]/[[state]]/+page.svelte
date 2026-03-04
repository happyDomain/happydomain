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
    import { Col, Container, Row } from "@sveltestrap/sveltestrap";

    import ProviderSidebar from "$lib/components/providers/Sidebar.svelte";
    import ProviderFormPage from "$lib/components/pages/Provider.svelte";
    import { providers, refreshProviders } from "$lib/stores/providers";

    interface Props {
        data: { ptype: string; state: number; providerId?: string };
    }

    let { data }: Props = $props();
    if (!$providers) refreshProviders();
</script>

<Container fluid class="d-flex flex-column flex-fill">
    <Row class="flex-fill">
        <Col
            sm={4}
            md={3}
            class="py-3 sticky-top d-flex flex-column"
            style="background-color: #edf5f2; overflow-y: auto; max-height: 100vh; z-index: 0"
        >
            <ProviderSidebar currentProviderId={data.providerId} />
        </Col>
        <Col sm={8} md={9} class="d-flex flex-column">
            <ProviderFormPage ptype={data.ptype} state={data.state} providerId={data.providerId} />
        </Col>
    </Row>
</Container>
