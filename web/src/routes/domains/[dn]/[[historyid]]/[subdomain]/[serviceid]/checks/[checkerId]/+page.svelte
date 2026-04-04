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
    import { page } from "$app/state";

    import { t } from "$lib/translations";
    import type { Domain } from "$lib/model/domain";
    import { fqdn } from "$lib/dns";
    import { domainLink } from "$lib/stores/domains";
    import CheckerConfigPage from "$lib/components/checkers/CheckerConfigPage.svelte";

    let domain: Domain = $derived(page.data.domain);
    let zoneId: string = $derived(page.data.zoneId);
    let subdomain: string = $derived(page.data.subdomain);
    let serviceid: string = $derived(page.data.serviceid);
    let checkerId = $derived(page.params.checkerId!);
    let checksBase = $derived(
        `/domains/${domainLink(domain.id)}/${encodeURIComponent(zoneId)}/${encodeURIComponent(page.params.subdomain!)}/${encodeURIComponent(serviceid)}/checks`,
    );
</script>

<CheckerConfigPage
    scope={{ domainId: domain.id, zoneId, subdomain, serviceId: serviceid }}
    {checksBase}
    {checkerId}
    domainName={fqdn(subdomain, domain.domain)}
    editableGroups={(status) => [
        {
            label: $t("checkers.option-groups.service-settings"),
            opts: status.options?.serviceOpts || [],
        },
        {
            label: $t("checkers.detail.admin-options"),
            opts: status.options?.adminOpts || [],
        },
        {
            label: $t("checkers.detail.configuration"),
            opts: status.options?.userOpts || [],
        },
    ]}
    readOnlyGroups={(status) => [
        {
            key: "domainOpts",
            label: $t("checkers.option-groups.domain-settings"),
            opts: status.options?.domainOpts || [],
        },
        {
            key: "runOpts",
            label: $t("checkers.option-groups.checker-parameters"),
            opts: status.options?.runOpts || [],
        },
    ]}
/>
