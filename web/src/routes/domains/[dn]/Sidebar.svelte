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
    import { domainCheckerLinks, serviceCheckerLinks } from "$links";
    import { resolve } from "$app/paths";
    import { page } from "$app/state";

    import { Button, Icon, Spinner } from "@sveltestrap/sveltestrap";

    import ChecksSidebarContent from "$lib/components/checkers/ChecksSidebarContent.svelte";
    import LogDaysTimeline from "$lib/components/domains/LogDaysTimeline.svelte";
    import SelectDomain from "$lib/components/domains/SelectDomain.svelte";
    import { groupLogsByDay, isProblem } from "$lib/domain_logs";
    import type { Domain } from "$lib/model/domain";
    import type { ZoneMeta } from "$lib/model/zone";
    import { domainLink, domains_idx } from "$lib/stores/domains";
    import { thisZone } from "$lib/stores/thiszone";
    import { useZoneEditor } from "$lib/components/zones/editor";
    import { t } from "$lib/translations";
    import { isReverseZone } from "$lib/dns";
    import { sortedDomains, sortedDomainsWithIntermediate } from "$lib/stores/thiszone";
    import ButtonZonePublish from "./ButtonZonePublish.svelte";
    import HistoryTimeline from "$lib/components/zones/HistoryTimeline.svelte";
    import ServiceSidebar from "$lib/components/zones/ServiceSidebar.svelte";
    import SubdomainListTiny from "$lib/components/zones/SubdomainListTiny.svelte";
    import DashboardSidebar from "./DashboardSidebar.svelte";
    import DomainActionsMenu from "./DomainActionsMenu.svelte";
    import OverviewLink from "./OverviewLink.svelte";
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

    const editor = useZoneEditor(() => domain);

    let isZonePage = $derived(
        page.route.id === "/domains/[dn]" || page.route.id === "/domains/[dn]/zone/[subdomain]",
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

    // The dashboard goes up to the list of domains; the zone, back to the
    // dashboard; any other page, to its parent.
    let isDashboard = $derived(page.route.id === "/domains/[dn]");
    let backHref = $derived(
        isDashboard
            ? resolve("/domains")
            : page.route.id === "/domains/[dn]/zone/[subdomain]"
              ? resolve("/domains/[dn]", { dn: encodeURIComponent(domainLink(selectedDomain)) })
              : "..",
    );
</script>

{#if $domains_idx[selectedDomain]}
    <!-- Header: back button + domain selector (always visible) -->
    <div class="d-flex">
        <Button href={backHref} class="fw-bolder" color="link">
            <Icon name={isDashboard ? "chevron-up" : "chevron-left"} />
        </Button>
        <SelectDomain bind:selectedDomain />
    </div>

    <div class="d-flex align-items-stretch gap-1 pt-2 pb-1">
        <OverviewLink {domain} active={isDashboard} />
        <DomainActionsMenu
            origin={domain}
            {selectedDomain}
            {selectedHistory}
            {onretrieveZoneDone}
        />
    </div>

    <!-- Main content: routed sidebar (scrolls along with the Col itself) -->
    {#if page.route.id && page.route.id.startsWith("/domains/[dn]/checkers")}
        <ChecksSidebarContent
            {domain}
            links={domainCheckerLinks(encodeURIComponent(domainLink(selectedDomain)))}
        />
    {:else if page.route.id === "/domains/[dn]/history/[zid]/zone/[subdomain]/[serviceid]"}
        <ServiceSidebar
            origin={domain}
            subdomain={page.data.subdomain ?? ""}
            serviceid={page.data.serviceid ?? ""}
            historyId={page.data.history ?? ""}
        />
    {:else if page.route.id && page.route.id.startsWith("/domains/[dn]/history/[zid]/zone")}
        <a
            href={resolve("/domains/[dn]/history/[zid]", {
                dn: encodeURIComponent(domainLink(selectedDomain)),
                zid: encodeURIComponent(page.params.zid ?? ""),
            })}
            class="sidebar-back d-flex align-items-center gap-1 mt-3 text-muted text-decoration-none fw-semibold"
        >
            <Icon name="chevron-left" />
            {$t("history.title")}
        </a>
        <div class="min-h-0 overflow-y-auto mt-2">
            {#if $sortedDomains && $sortedDomainsWithIntermediate && $thisZone}
                {@const zoneId = $thisZone.id}
                <SubdomainListTiny
                    domains={isReverseZone(domain.domain)
                        ? $sortedDomains
                        : $sortedDomainsWithIntermediate}
                    origin={domain}
                    href={(dn) => editor.subdomainHref(zoneId, dn)}
                    current={page.data.subdomain}
                />
            {/if}
        </div>
    {:else if page.route.id && page.route.id.startsWith("/domains/[dn]/history")}
        {#if page.data.historyDomain}
            <div class="min-h-0 overflow-y-auto">
                <HistoryTimeline
                    domain={page.data.historyDomain}
                    selected={page.params.zid}
                    dn={encodeURIComponent(domainLink(selectedDomain))}
                />
            </div>
        {/if}
    {:else if page.route.id && page.route.id.startsWith("/domains/[dn]/logs")}
        {#if page.data.logs}
            {@const onlyProblems = page.url.searchParams.get("only") === "problems"}
            <div class="min-h-0 overflow-y-auto">
                <LogDaysTimeline
                    days={groupLogsByDay(
                        onlyProblems ? page.data.logs.filter(isProblem) : page.data.logs,
                    )}
                    {onlyProblems}
                    filterHref={(only) =>
                        resolve("/domains/[dn]/logs", {
                            dn: encodeURIComponent(domainLink(selectedDomain)),
                        }) + (only ? "?only=problems" : "")}
                />
            </div>
        {/if}
    {:else if page.route.id && page.route.id.startsWith("/domains/[dn]/export")}
        <div class="min-h-0 overflow-y-auto mt-2">
            {#if $sortedDomains && $sortedDomainsWithIntermediate && $thisZone}
                {@const zoneId = $thisZone.id}
                <SubdomainListTiny
                    domains={isReverseZone(domain.domain)
                        ? $sortedDomains
                        : $sortedDomainsWithIntermediate}
                    origin={domain}
                    href={(dn) => editor.subdomainHref(zoneId, dn)}
                />
            {/if}
        </div>
    {:else if page.route.id && page.route.id.startsWith("/domains/[dn]/zone/[subdomain]/[serviceid]/checkers")}
        <ChecksSidebarContent
            {domain}
            links={serviceCheckerLinks(
                encodeURIComponent(domainLink(selectedDomain)),
                encodeURIComponent(page.params.subdomain ?? ""),
                encodeURIComponent(page.data.serviceid ?? ""),
            )}
            serviceContext={{
                zoneId: page.data.zoneId ?? "",
                subdomain: page.data.subdomain ?? "",
                serviceid: page.data.serviceid ?? "",
            }}
        />
    {:else if page.route.id === "/domains/[dn]/zone/[subdomain]/[serviceid]"}
        <ServiceSidebar
            origin={domain}
            subdomain={page.data.subdomain ?? ""}
            serviceid={page.data.serviceid ?? ""}
            historyId={page.data.history ?? ""}
        />
    {:else if page.route.id === "/domains/[dn]"}
        <DashboardSidebar origin={domain} {selectedHistory} />
    {:else}
        <ZoneSidebar origin={domain} {selectedHistory} />
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
        background-color: var(--bs-secondary-bg);
        z-index: 10;
    }
    .sidebar-footer-spacer {
        flex-shrink: 0;
        height: 3.5rem;
    }
</style>
