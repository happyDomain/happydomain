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
    import { Icon } from "@sveltestrap/sveltestrap";

    import { worstStatus } from "$lib/domain_checks";
    import type { Domain } from "$lib/model/domain";
    import { useZoneEditor } from "$lib/components/zones/editor";
    import { checksBySubdomain } from "$lib/stores/domainchecks";
    import { sortedDomains, thisZone } from "$lib/stores/thiszone";
    import { t } from "$lib/translations";
    import { getStatusColor, StatusCrit, StatusError, StatusWarn } from "$lib/utils/checkers";

    interface Props {
        origin: Domain;
    }

    let { origin }: Props = $props();

    const editor = useZoneEditor(() => origin);

    let checks = $derived(
        $checksBySubdomain?.domainId === origin.id ? $checksBySubdomain.sections : undefined,
    );

    function suffix(dn: string): string {
        return dn === "" ? "" : "." + origin.domain;
    }

    // What the checks of a subdomain found; empty when none ran.
    function note(dn: string): string {
        const items = checks?.get(dn)?.items ?? [];
        let crit = 0;
        let warn = 0;
        let ran = 0;
        for (const item of items) {
            const code = item.status.latestExecution?.result?.status;
            if (code === undefined || code === null) continue;
            ran++;
            if (code === StatusCrit || code === StatusError) crit++;
            else if (code === StatusWarn) warn++;
        }
        if (crit + warn === 0) return ran > 0 ? $t("domains.dashboard.subdomains.all-pass") : "";
        return [
            crit > 0 && `${crit} ${$t("checkers.dashboard.critical")}`,
            warn > 0 && `${warn} ${$t("checkers.dashboard.needs-attention")}`,
        ]
            .filter(Boolean)
            .join(" · ");
    }
</script>

{#if $sortedDomains && $thisZone}
    {@const zoneId = $thisZone.id}
    <div class="list-group">
        {#each $sortedDomains as dn (dn)}
            {@const section = checks?.get(dn)}
            {@const nbServices = $thisZone.services[dn]?.length ?? 0}
            <a
                href={editor.subdomainHref(zoneId, dn)}
                class="list-group-item list-group-item-action d-flex align-items-center gap-3 py-3"
            >
                <span
                    class="d-inline-block flex-shrink-0 p-1 rounded-circle bg-{section
                        ? getStatusColor(worstStatus(section.items))
                        : 'secondary-subtle'}"
                ></span>
                <span class="col-6 col-md-4 text-truncate font-monospace small fw-medium">
                    {dn === "" ? "@" : dn}<span class="fw-normal text-body-tertiary"
                        >{suffix(dn)}</span
                    >
                </span>
                <span class="col-3 col-md-2 small text-body-secondary">
                    {$t("domains.n-services", { count: nbServices })}
                </span>
                <span class="flex-fill text-truncate small text-body-tertiary d-none d-md-inline">
                    {note(dn)}
                </span>
                <Icon name="chevron-right" class="ms-auto text-body-tertiary" />
            </a>
        {/each}
    </div>
{/if}
