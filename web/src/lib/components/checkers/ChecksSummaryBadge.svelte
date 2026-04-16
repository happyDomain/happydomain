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
    import { Badge, Icon } from "@sveltestrap/sveltestrap";

    import type { HappydnsStatus } from "$lib/api-base/types.gen";
    import { getStatusColor, getStatusIcon, getStatusI18nKey } from "$lib/utils";
    import { t } from "$lib/translations";

    interface Props {
        status: HappydnsStatus | undefined;
        checksBase: string;
    }

    let { status, checksBase }: Props = $props();
</script>

{#if status !== undefined}
    <a
        href={checksBase}
        class="text-decoration-none"
        title={$t("domains.actions.checks")}
        aria-label={$t("domains.actions.checks")}
    >
        <Badge color={getStatusColor(status)}>
            <Icon name={getStatusIcon(status)} class="me-1" />
            {$t(getStatusI18nKey(status))}
        </Badge>
    </a>
{/if}
