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
    import { Alert, Badge, Button, Icon } from "@sveltestrap/sveltestrap";

    import DashboardSection from "$lib/components/DashboardSection.svelte";
    import PageTitle from "$lib/components/PageTitle.svelte";
    import { useZoneEditor } from "$lib/components/zones/editor";
    import {
        countServices,
        isGlobalTemplate,
        templateOrigin,
        type ZoneTemplate,
    } from "$lib/model/zone_template";
    import { sortedDomains, thisZone } from "$lib/stores/thiszone";
    import { thisTemplate } from "$lib/stores/zone_templates";
    import { t } from "$lib/translations";

    interface Props {
        template: ZoneTemplate;
        // Nothing can be changed, as a global template shown to a user.
        readOnly: boolean;
    }

    let { template: loaded, readOnly }: Props = $props();

    let template = $derived($thisTemplate?.id === loaded.id ? $thisTemplate : loaded);
    const origin = templateOrigin();
    const editor = useZoneEditor(() => origin);

    // An empty template still has its apex, to add services to.
    let subdomains = $derived($sortedDomains?.length ? $sortedDomains : [""]);

    const dateFmt = new Intl.DateTimeFormat(undefined, {
        dateStyle: "medium",
        timeStyle: "short",
    });
</script>

<div style="max-width: 100%;" class="w-100 pt-1 mb-5">
    <PageTitle title={template.name} subtitle={template.description || undefined}>
        {#snippet undertitle()}
            {#if isGlobalTemplate(template)}
                <Badge color="info" class="mt-2" title={$t("zone-templates.global-hint")}>
                    <Icon name="globe" />
                    {$t("zone-templates.global")}
                </Badge>
            {/if}
        {/snippet}
        <div class="d-flex flex-column align-items-end gap-2">
            <Button
                color="primary"
                class="fw-semibold"
                href={editor.subdomainHref(template.id, "")}
            >
                {readOnly ? $t("zone-templates.actions.view") : $t("domains.edit-records")}
                <Icon name="chevron-right" />
            </Button>
            <dl class="vstack gap-1 align-items-end small mb-0">
                <div class="d-flex gap-3">
                    <dt class="fw-normal text-body-secondary">
                        {$t("zone-templates.services-title")}
                    </dt>
                    <dd class="fw-medium text-body-emphasis mb-0">
                        {countServices(template)}
                    </dd>
                </div>
                {#if template.last_modified}
                    <div class="d-flex gap-3">
                        <dt class="fw-normal text-body-secondary">
                            {$t("zone-templates.last-modified")}
                        </dt>
                        <dd class="fw-medium text-body-emphasis mb-0">
                            {dateFmt.format(new Date(template.last_modified))}
                        </dd>
                    </div>
                {/if}
            </dl>
        </div>
    </PageTitle>

    {#if readOnly && isGlobalTemplate(template)}
        <Alert color="info" fade={false}>
            <Icon name="globe" />
            {$t("zone-templates.global-hint")}
        </Alert>
    {/if}

    {#if $thisZone}
        {@const zone = $thisZone}
        <DashboardSection
            title={$t("domains.dashboard.subdomains.title")}
            subtitle={$t("domains.dashboard.subdomains.subtitle", { count: subdomains.length })}
        >
            <div class="list-group">
                {#each subdomains as dn (dn)}
                    {@const services = zone.services[dn] ?? []}
                    <a
                        href={editor.subdomainHref(template.id, dn)}
                        class="list-group-item list-group-item-action d-flex align-items-center gap-3 py-3"
                    >
                        <span class="col-6 col-md-4 text-truncate font-monospace small fw-medium">
                            {dn === "" ? "@" : dn}<span class="fw-normal text-body-tertiary"
                                >{dn === "" ? "" : "." + origin.domain}</span
                            >
                        </span>
                        <span class="col-3 col-md-2 small text-body-secondary">
                            {$t("domains.n-services", { count: services.length })}
                        </span>
                        <span class="flex-fill small text-body-tertiary d-none d-md-inline">
                            {$t("domains.n-records", {
                                count: services.reduce((n, svc) => n + (svc._tmp_hint_nb ?? 0), 0),
                            })}
                        </span>
                        <Icon name="chevron-right" class="ms-auto text-body-tertiary" />
                    </a>
                {/each}
            </div>
        </DashboardSection>
    {/if}
</div>
