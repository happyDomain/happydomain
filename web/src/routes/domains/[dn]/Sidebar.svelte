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
    import { page } from "$app/state";

    import { Button, Icon, Spinner } from "@sveltestrap/sveltestrap";

    import ChecksSidebarContent from "$lib/components/checkers/ChecksSidebarContent.svelte";
    import SelectDomain from "$lib/components/domains/SelectDomain.svelte";
    import type { Domain } from "$lib/model/domain";
    import type { ZoneMeta } from "$lib/model/zone";
    import { domainLink, domains_idx } from "$lib/stores/domains";
    import { thisZone } from "$lib/stores/thiszone";
    import { t } from "$lib/translations";
    import ButtonZonePublish from "./ButtonZonePublish.svelte";
    import ServiceSidebar from "./ServiceSidebar.svelte";
    import ZoneSidebar from "./ZoneSidebar.svelte";

    interface Props {
        domain: Domain;
        selectedDomain: string;
        selectedHistory: string | undefined;
        deleteInProgress: boolean;
        ondetachClick: () => void;
        onretrieveZoneDone: (zm: ZoneMeta) => void;
    }

    let {
        domain,
        selectedDomain = $bindable(),
        selectedHistory,
        deleteInProgress,
        ondetachClick,
        onretrieveZoneDone,
    }: Props = $props();

    let isZonePage = $derived(
        page.route.id === "/domains/[dn]" || page.route.id === "/domains/[dn]/[[historyid]]",
    );
    let showPublishFooter = $derived(
        isZonePage &&
            !!domain.zone_history &&
            !!$domains_idx[selectedDomain] &&
            domain.id === $domains_idx[selectedDomain]?.id &&
            !!selectedHistory &&
            !!$thisZone,
    );
    let showDetachFooter = $derived(isZonePage && !showPublishFooter);
</script>

{#if $domains_idx[selectedDomain]}
    <!-- Header: back button + domain selector (always visible) -->
    <div class="d-flex">
        <Button href={isZonePage ? "/domains/" : ".."} class="fw-bolder" color="link">
            <Icon name={isZonePage ? "chevron-up" : "chevron-left"} />
        </Button>
        <SelectDomain bind:selectedDomain />
    </div>

    <!-- Main content: routed sidebar (scrolls along with the Col itself) -->
    {#if page.route.id && page.route.id.startsWith("/domains/[dn]/checkers")}
        <ChecksSidebarContent
            {domain}
            checksBase={"/domains/" + encodeURIComponent(domainLink(selectedDomain)) + "/checkers"}
            backHref={"/domains/" + encodeURIComponent(domainLink(selectedDomain))}
        />
    {:else if page.route.id && (page.route.id.startsWith("/domains/[dn]/checks") || page.route.id.startsWith("/domains/[dn]/history") || page.route.id.startsWith("/domains/[dn]/logs") || page.route.id.startsWith("/domains/[dn]/[[historyid]]/export"))}
        <a
            href="/domains/{encodeURIComponent(domainLink(selectedDomain))}"
            class="sidebar-back d-flex align-items-center gap-1 mt-3 text-muted text-decoration-none fw-semibold"
        >
            <Icon name="chevron-left" />
            {$t("zones.return-to")}
        </a>
    {:else if page.route.id && page.route.id.startsWith("/domains/[dn]/[[historyid]]/[subdomain]/[serviceid]/checks")}
        <a
            href={"/domains/" +
                encodeURIComponent(domainLink(selectedDomain)) +
                "/" +
                encodeURIComponent(page.data.history ?? "") +
                "/" +
                encodeURIComponent(page.params.subdomain ?? "") +
                "/" +
                encodeURIComponent(page.data.serviceid ?? "")}
            class="sidebar-back d-flex align-items-center gap-1 mt-3 text-muted text-decoration-none fw-semibold"
        >
            <Icon name="chevron-left" />
            {$t("zones.return-to")}
        </a>
    {:else if page.route.id && page.route.id.startsWith("/domains/[dn]/[[historyid]]/[subdomain]/[serviceid]/checkers")}
        <ChecksSidebarContent
            {domain}
            checksBase={"/domains/" +
                encodeURIComponent(domainLink(selectedDomain)) +
                "/" +
                encodeURIComponent(page.data.history ?? "") +
                "/" +
                encodeURIComponent(page.params.subdomain ?? "") +
                "/" +
                encodeURIComponent(page.data.serviceid ?? "") +
                "/checkers"}
            backHref={"/domains/" +
                encodeURIComponent(domainLink(selectedDomain)) +
                "/" +
                encodeURIComponent(page.data.history ?? "") +
                "/" +
                encodeURIComponent(page.params.subdomain ?? "") +
                "/" +
                encodeURIComponent(page.data.serviceid ?? "")}
            serviceContext={{
                zoneId: page.data.zoneId ?? "",
                subdomain: page.data.subdomain ?? "",
                serviceid: page.data.serviceid ?? "",
            }}
        />
    {:else if page.route.id === "/domains/[dn]/[[historyid]]/[subdomain]/[serviceid]"}
        <ServiceSidebar
            origin={domain}
            subdomain={page.data.subdomain ?? ""}
            serviceid={page.data.serviceid ?? ""}
            historyId={page.data.history ?? ""}
        />
    {:else}
        <ZoneSidebar origin={domain} {selectedDomain} {selectedHistory} {onretrieveZoneDone} />
    {/if}

    <!-- Spacer so the last sidebar item isn't hidden behind the fixed footer -->
    {#if showPublishFooter || showDetachFooter}
        <div class="sidebar-footer-spacer"></div>
    {/if}
{:else}
    <div class="mt-4 text-center">
        <Spinner color="primary" />
    </div>
{/if}

<!-- Fixed footer pinned to the bottom of the viewport, matching sidebar width -->
{#if $domains_idx[selectedDomain] && selectedHistory && (showPublishFooter || showDetachFooter)}
    <div class="sidebar-footer-fixed col-sm-4 col-md-3">
        {#if showPublishFooter}
            <ButtonZonePublish
                class="w-100 border-top border-muted pt-2"
                {domain}
                history={selectedHistory}
            />
        {:else}
            <Button
                color="danger"
                outline
                class="w-100"
                disabled={deleteInProgress}
                on:click={ondetachClick}
            >
                {#if deleteInProgress}
                    <Spinner size="sm" />
                {:else}
                    <Icon name="trash" />
                {/if}
                {$t("domains.stop")}
            </Button>
        {/if}
    </div>
{/if}

<style>
    .sidebar-footer-fixed {
        position: fixed;
        bottom: 0;
        left: 0;
        padding: 0.75rem;
        background-color: #edf5f2;
        z-index: 10;
    }
    .sidebar-footer-spacer {
        flex-shrink: 0;
        height: 3.5rem;
    }
</style>
