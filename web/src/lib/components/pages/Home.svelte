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
    import {
        Col,
        Container,
        Row,
    } from "@sveltestrap/sveltestrap";

    import { createDomain } from "$lib/api/provider";
    import DomainListSection from "$lib/components/pages/home/DomainListSection.svelte";
    import Logo from "$lib/components/Logo.svelte";
    import Sidebar from "$lib/components/pages/home/Sidebar.svelte";
    import { domains, refreshDomains } from "$lib/stores/domains";
    import {
        providers,
        providersSpecs,
        refreshProviders,
        refreshProvidersSpecs,
    } from "$lib/stores/providers";
    import { t } from "$lib/translations";

    if (!$domains) refreshDomains();
    if (!$providers) refreshProviders();
    if (!$providersSpecs) refreshProvidersSpecs();
</script>

<Container class="flex-fill pt-4 pb-5">
    <h1 class="text-center mb-4">
        {$t("common.welcome.start")}<Logo height="40" />{$t("common.welcome.end")}
    </h1>

    <Row>
        <Col md="8" class="order-1 order-md-0">
            <DomainListSection />
        </Col>

        <Col md="4" class="order-0 order-md-1">
            <Sidebar />
        </Col>
    </Row>
</Container>
