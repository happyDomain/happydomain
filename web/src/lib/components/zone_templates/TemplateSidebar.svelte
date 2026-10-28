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
    import { templateLinks } from "$links";

    import {
        Button,
        Dropdown,
        DropdownItem,
        DropdownMenu,
        DropdownToggle,
        Icon,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { useZoneEditor } from "$lib/components/zones/editor";
    import { controls as ctrlNewSubdomain } from "$lib/components/zones/NewSubdomainPath.svelte";
    import ServiceSidebar from "$lib/components/zones/ServiceSidebar.svelte";
    import SubdomainListTiny from "$lib/components/zones/SubdomainListTiny.svelte";
    import { templateOrigin, type ZoneTemplate } from "$lib/model/zone_template";
    import { sortedDomains, sortedDomainsWithIntermediate } from "$lib/stores/thiszone";
    import { zoneTemplates } from "$lib/stores/zone_templates";
    import { t } from "$lib/translations";
    import { controls as ctrlTemplateDetails } from "./ModalTemplateDetails.svelte";
    import SelectTemplate from "./SelectTemplate.svelte";

    interface Props {
        template: ZoneTemplate;
        // Nothing can be changed, as a global template shown to a user.
        readOnly: boolean;
        // The subdomain of the page, "" being the apex; none on the overview.
        subdomain?: string;
        // The service of the page, under subdomain.
        serviceid?: string;
        deleteInProgress: boolean;
        onselect: (id: string) => void;
        ondelete: () => void;
    }

    let { template, readOnly, subdomain, serviceid, deleteInProgress, onselect, ondelete }: Props =
        $props();

    const origin = templateOrigin();
    const editor = useZoneEditor(() => origin);

    let overviewHref = $derived(templateLinks().template(encodeURIComponent(template.id)));
    let isOverview = $derived(subdomain === undefined);

    // The overview goes up to the list of templates; a subdomain, back to the
    // overview; a service, back to its subdomain.
    let back = $derived(
        subdomain === undefined
            ? { href: templateLinks().list, title: $t("zone-templates.back") }
            : serviceid === undefined
              ? { href: overviewHref, title: $t("zone-templates.overview") }
              : {
                    href: editor.subdomainHref(template.id, subdomain),
                    title: $t("zones.return-to"),
                },
    );
</script>

<!-- Header: back button + template selector -->
<div class="d-flex">
    <Button href={back.href} class="fw-bolder" color="link" title={back.title}>
        <Icon name={isOverview ? "chevron-up" : "chevron-left"} />
    </Button>
    <SelectTemplate templates={$zoneTemplates} current={template} {onselect} />
</div>

<div class="d-flex align-items-stretch gap-1 pt-2 pb-1">
    <a
        href={overviewHref}
        class="btn d-flex flex-fill align-items-center gap-2 p-2 text-start lh-sm overflow-hidden"
        class:btn-outline-dark={!isOverview}
        class:text-bg-dark={isOverview}
        aria-current={isOverview ? "page" : undefined}
    >
        <Icon name="house-door-fill" class="flex-shrink-0" />
        <span class="flex-fill overflow-hidden">
            <span class="d-block fw-semibold text-truncate">{$t("zone-templates.overview")}</span>
            <span class="d-block small font-monospace text-truncate opacity-75">
                {$t("zone-templates.overview-hint")}
            </span>
        </span>
    </a>
    <Dropdown class="d-flex flex-shrink-0" size="sm">
        <DropdownToggle
            color="secondary"
            outline
            size="sm"
            class="d-flex flex-column align-items-center justify-content-center gap-1 lh-sm text-nowrap"
        >
            {#if deleteInProgress}
                <Spinner size="sm" />
            {:else}
                <Icon name="gear-fill" aria-hidden="true" />
            {/if}
            <span class="small">{$t("domains.actions.label")}</span>
        </DropdownToggle>
        <DropdownMenu end style="z-index: 11;">
            <DropdownItem header>
                {$t("zone-templates.title")}
                <span class="d-block fw-normal text-muted text-truncate">{template.name}</span>
            </DropdownItem>
            <DropdownItem
                disabled={readOnly}
                title={readOnly ? $t("zone-templates.global-hint") : undefined}
                on:click={() => ctrlTemplateDetails.Open()}
            >
                {$t("zone-templates.edit")}
            </DropdownItem>
            <DropdownItem
                disabled={readOnly || deleteInProgress}
                title={readOnly ? $t("zone-templates.global-hint") : undefined}
                on:click={ondelete}
            >
                {$t("zone-templates.delete")}
            </DropdownItem>
        </DropdownMenu>
    </Dropdown>
</div>

{#if subdomain !== undefined && serviceid !== undefined}
    <ServiceSidebar {origin} {subdomain} {serviceid} historyId={template.id} />
{:else}
    <div
        class="d-flex align-items-center gap-1 pt-3 pb-1 small fw-semibold text-uppercase text-body-secondary"
    >
        <span class="flex-fill text-truncate">
            {$t("domains.sidebar.subdomains")}
        </span>
        <!-- A disabled button gets no pointer events: its wrapper carries the tooltip. -->
        <span
            class="flex-shrink-0"
            title={readOnly ? $t("zone-templates.global-hint") : $t("domains.add-a-subdomain")}
        >
            <Button
                type="button"
                color="link"
                size="sm"
                class="text-body-secondary p-0 lh-1"
                aria-label={$t("domains.add-a-subdomain")}
                disabled={!$sortedDomains || readOnly}
                on:click={() => ctrlNewSubdomain.Open()}
            >
                <Icon name="plus-lg" />
            </Button>
        </span>
    </div>
    <div style="min-height:0; overflow-y: auto;">
        {#if $sortedDomains && $sortedDomainsWithIntermediate}
            <SubdomainListTiny
                domains={$sortedDomains.length ? $sortedDomainsWithIntermediate : [""]}
                {origin}
                href={(dn) => editor.subdomainHref(template.id, dn)}
                current={subdomain}
            />
        {/if}
    </div>
{/if}
