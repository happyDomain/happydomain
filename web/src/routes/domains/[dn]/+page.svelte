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
    import { domainCheckerLinks, domainLinks, serviceCheckerLinks } from "$links";
    import { page } from "$app/state";

    import { Button, Icon } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import type { CheckTarget } from "$lib/domain_checks";
    import type { Domain } from "$lib/model/domain";
    import { domainLink } from "$lib/stores/domains";
    import { domainChecks } from "$lib/stores/domainchecks";
    import CheckResultsDashboard from "$lib/components/checkers/CheckResultsDashboard.svelte";
    import ProviderLink from "$lib/components/providers/ProviderLink.svelte";
    import DashboardSection from "$lib/components/DashboardSection.svelte";
    import { sortedDomains } from "$lib/stores/thiszone";
    import SubdomainGlance from "./SubdomainGlance.svelte";

    let domain: Domain = $derived(page.data.domain);
    let sections = $derived(
        $domainChecks?.domainId === domain.id ? $domainChecks.sections : undefined,
    );
    const linksFor = (target?: CheckTarget) =>
        target
            ? serviceCheckerLinks(
                  domainLink(domain.id),
                  encodeURIComponent(target.subdomain),
                  encodeURIComponent(target.serviceId),
              )
            : domainCheckerLinks(domainLink(domain.id));

    // The most recent version of the zone sent to the provider.
    let lastPublished = $derived(
        domain.zone_history
            .map((id) => domain.zone_meta?.[id]?.published)
            .find((published) => published),
    );
    const publishFmt = new Intl.DateTimeFormat(undefined, {
        dateStyle: "medium",
        timeStyle: "short",
    });
</script>

<CheckResultsDashboard
    {sections}
    {linksFor}
    domainName={domain.domain}
    title={$t("checkers.list.title") + domain.domain}
>
    {#snippet summary()}
        <div class="d-flex flex-column align-items-end gap-2">
            <Button
                color="primary"
                class="fw-semibold"
                href={domainLinks().subdomain(encodeURIComponent(domainLink(domain.id)), "@")}
            >
                {$t("domains.edit-records")}
                <Icon name="chevron-right" />
            </Button>
            <dl class="vstack gap-1 align-items-end small mb-0">
                <div class="d-flex gap-3">
                    <dt class="fw-normal text-body-secondary">{$t("domains.meta.provider")}</dt>
                    <dd class="fw-medium text-body-emphasis mb-0">
                        <ProviderLink id_provider={domain.id_provider} />
                    </dd>
                </div>
                <div class="d-flex gap-3">
                    <dt class="fw-normal text-body-secondary">
                        {$t("domains.meta.last-publish")}
                    </dt>
                    <dd class="fw-medium text-body-emphasis mb-0">
                        {lastPublished
                            ? publishFmt.format(new Date(lastPublished))
                            : $t("domains.meta.never-published")}
                    </dd>
                </div>
            </dl>
        </div>
    {/snippet}

    {#if $sortedDomains}
        <DashboardSection
            title={$t("domains.dashboard.subdomains.title")}
            subtitle={$t("domains.dashboard.subdomains.subtitle", {
                count: $sortedDomains.length,
            })}
        >
            <SubdomainGlance origin={domain} />
        </DashboardSection>
    {/if}
</CheckResultsDashboard>
