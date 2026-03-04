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
    import { page } from "$app/stores";
    import { goto } from "$app/navigation";
    import { get } from "svelte/store";
    import { Col, Container, Row } from "@sveltestrap/sveltestrap";

    import DomainListSection from "$lib/components/pages/home/DomainListSection.svelte";
    import Logo from "$lib/components/Logo.svelte";
    import Sidebar from "$lib/components/pages/home/Sidebar.svelte";
    import { domains, refreshDomains } from "$lib/stores/domains";
    import { filteredGroup, filteredName, filteredProvider } from "$lib/stores/home";
    import {
        providers,
        providers_idx,
        providersSpecs,
        refreshProviders,
        refreshProvidersSpecs,
    } from "$lib/stores/providers";
    import { t } from "$lib/translations";

    // Initialize filter stores from URL query params
    const searchParams = $page.url.searchParams;
    filteredName.set(searchParams.get("name") || "");
    filteredGroup.set(searchParams.has("group") ? searchParams.get("group") : null);

    const initialProviderId = searchParams.get("provider");
    let pendingProviderId: string | null = initialProviderId;
    if (initialProviderId) {
        const idx = get(providers_idx);
        if (idx[initialProviderId]) {
            filteredProvider.set(idx[initialProviderId]);
            pendingProviderId = null;
        } else {
            filteredProvider.set(null);
            // Wait for providers to load, then resolve
            const unsubscribe = providers.subscribe((provs) => {
                if (provs !== undefined) {
                    const found = provs.find((p) => p._id === initialProviderId);
                    if (found) filteredProvider.set(found);
                    unsubscribe();
                }
            });
        }
    } else {
        filteredProvider.set(null);
    }

    if (!$domains) refreshDomains();
    if (!$providers) refreshProviders();
    if (!$providersSpecs) refreshProvidersSpecs();

    // Sync filter stores to URL query params
    $effect(() => {
        const params = new URLSearchParams();
        if ($filteredName) params.set("name", $filteredName);
        if ($filteredProvider) {
            params.set("provider", $filteredProvider._id);
            pendingProviderId = null;
        } else if (pendingProviderId) {
            params.set("provider", pendingProviderId);
        }
        if ($filteredGroup !== null) params.set("group", $filteredGroup ?? "");

        const newSearch = params.toString();
        const currentSearch = window.location.search.replace(/^\?/, "");
        if (newSearch !== currentSearch) {
            const newUrl = window.location.pathname + (newSearch ? `?${newSearch}` : "");
            goto(newUrl, { replaceState: true, keepFocus: true, noScroll: true });
        }
    });
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
