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
    import { resolve } from "$app/paths";
    import { Icon } from "@sveltestrap/sveltestrap";

    import type { Domain } from "$lib/model/domain";
    import { domainLink } from "$lib/stores/domains";
    import { t } from "$lib/translations";

    interface Props {
        domain: Domain;
        // The overview is the page shown.
        active?: boolean;
    }

    let { domain, active = false }: Props = $props();
</script>

<a
    href={resolve("/domains/[dn]", { dn: encodeURIComponent(domainLink(domain.id)) })}
    class="btn d-flex flex-fill align-items-center gap-2 p-2 text-start lh-sm overflow-hidden"
    class:btn-outline-dark={!active}
    class:text-bg-dark={active}
    aria-current={active ? "page" : undefined}
>
    <Icon name="house-door-fill" class="flex-shrink-0" />
    <span class="flex-fill overflow-hidden">
        <span class="d-block fw-semibold text-truncate">{$t("domains.sidebar.overview")}</span>
        <span class="d-block small font-monospace text-truncate opacity-75">
            {$t("domains.sidebar.overview-hint")}
        </span>
    </span>
</a>
