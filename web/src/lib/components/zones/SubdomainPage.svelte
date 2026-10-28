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
    import { Alert, Button, Icon } from "@sveltestrap/sveltestrap";

    import AliasModal from "$lib/components/modals/Alias.svelte";
    import PageTitle from "$lib/components/PageTitle.svelte";
    import { controls as ctrlNewService } from "$lib/components/services/NewServicePath.svelte";
    import UserResource from "$lib/components/zones/UserResource.svelte";
    import { useZoneEditor } from "$lib/components/zones/editor";
    import type { Domain } from "$lib/model/domain";
    import { checksBySubdomain } from "$lib/stores/domainchecks";
    import { isReadOnlyProvider, providers_idx } from "$lib/stores/providers";
    import { thisZone } from "$lib/stores/thiszone";
    import { t } from "$lib/translations";
    import { StatusCrit, StatusError, StatusInfo, StatusOK, StatusWarn } from "$lib/utils/checkers";

    interface Props {
        domain: Domain;
        // "" for the apex.
        subdomain: string;
        // A past version of the zone, shown read-only.
        historical?: boolean;
    }

    let { domain, subdomain, historical = false }: Props = $props();

    let services = $derived($thisZone?.services[subdomain] ?? []);
    let nbRecords = $derived(services.reduce((n, svc) => n + (svc._tmp_hint_nb ?? 0), 0));

    const editor = useZoneEditor(() => domain);
    let readOnly = $derived(
        editor.readOnly || isReadOnlyProvider($providers_idx, domain.id_provider),
    );

    // What the checks of the subdomain found; they are about the live zone.
    let checks = $derived.by(() => {
        if (historical || $checksBySubdomain?.domainId !== domain.id) return undefined;
        const items = $checksBySubdomain.sections.get(subdomain)?.items;
        if (!items?.length) return undefined;
        let failing = 0;
        let warning = 0;
        let passing = 0;
        for (const item of items) {
            const code = item.status.latestExecution?.result?.status;
            if (code === StatusCrit || code === StatusError) failing++;
            else if (code === StatusWarn) warning++;
            else if (code === StatusOK || code === StatusInfo) passing++;
        }
        return { failing, warning, passing, total: items.length };
    });
</script>

{#if $thisZone}
    <div class="w-100 mw-100 pt-1 mb-5">
        <PageTitle classTitle="font-monospace">
            {#snippet title()}
                {#if subdomain}
                    {subdomain}<span class="text-body-tertiary fw-normal">.{domain.domain}</span>
                {:else}
                    {domain.domain}
                {/if}
            {/snippet}
            {#snippet undertitle()}
                <div
                    class="subdomain-stats d-flex flex-wrap align-items-center mt-2 text-body-secondary"
                >
                    <span class="fw-semibold text-body-emphasis">
                        {$t("domains.n-services", { count: services.length })}
                    </span>
                    <span class="text-body-tertiary">·</span>
                    <span>{$t("domains.n-records", { count: nbRecords })}</span>
                    {#if checks}
                        <span class="text-body-tertiary">·</span>
                        <span
                            class="checks-summary d-inline-flex align-items-center fw-semibold"
                            class:text-danger={checks.failing > 0}
                            class:text-warning-emphasis={checks.failing === 0 && checks.warning > 0}
                            class:text-success={checks.failing === 0 && checks.warning === 0}
                        >
                            <span class="checks-dot rounded-circle"></span>
                            {[
                                checks.failing > 0 &&
                                    $t("domains.checks.failing", { count: checks.failing }),
                                checks.warning > 0 &&
                                    $t("domains.checks.warning", { count: checks.warning }),
                                $t("domains.checks.passing", {
                                    count: checks.passing,
                                    total: checks.total,
                                }),
                            ]
                                .filter(Boolean)
                                .join(" · ")}
                        </span>
                    {/if}
                </div>
            {/snippet}
            {#if historical}
                <span class="badge text-bg-warning">
                    <Icon name="clock-history" />
                    {$t("history.title")}
                </span>
            {:else if !readOnly}
                <Button
                    type="button"
                    color="secondary"
                    outline
                    class="fw-semibold"
                    on:click={() => ctrlNewService.Open(subdomain)}
                >
                    <Icon name="plus-circle" />
                    {$t("domains.add-a-service")}
                </Button>
            {/if}
        </PageTitle>
        {#key subdomain}
            <UserResource
                dn={subdomain ? subdomain : "@"}
                origin={domain}
                {services}
                zoneId={$thisZone.id}
            />
        {/key}
        {#if $thisZone.partial}
            <Alert color="info" fade={false} class="mx-2">
                <Icon name="binoculars" />
                <strong>{$t("zones.partial.title")}</strong>
                {$t("zones.partial.description")}
            </Alert>
        {/if}
    </div>
{/if}

<AliasModal origin={domain} />

<style>
    .subdomain-stats {
        gap: 0.75rem;
        font-size: 0.8rem;
    }

    .checks-summary {
        gap: 0.4rem;
    }

    .checks-dot {
        width: 0.45rem;
        height: 0.45rem;
        background: currentColor;
    }
</style>
