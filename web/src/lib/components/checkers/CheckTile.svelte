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
    import type { HappydnsStatus } from "$lib/api-base/types.gen";
    import { Icon } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import { formatCheckDate, getStatusColor, getStatusI18nKey, getStatusIcon } from "$lib/utils";
    import { StatusCrit, StatusError, StatusWarn } from "$lib/utils/checkers";

    interface Props {
        status: HappydnsStatus | undefined;
        name: string;
        // The first message of the last run, summing it up.
        message?: string;
        startedAt?: Date | string;
        href?: string;
    }

    let { status, name, message, startedAt, href }: Props = $props();

    let alarming = $derived(
        status === StatusWarn || status === StatusCrit || status === StatusError,
    );
    let color = $derived(getStatusColor(status));
</script>

<a
    {href}
    class="card card-body h-100 d-flex flex-column gap-2 text-reset text-decoration-none border-start-{color} {alarming
        ? `border-${color}`
        : ''}"
>
    <div class="d-flex align-items-center gap-2">
        <span
            class="d-flex flex-shrink-0 p-1 lh-1 rounded-circle bg-{color}-subtle text-{color}-emphasis"
        >
            <Icon name={getStatusIcon(status)} />
        </span>
        <span class="small fw-semibold text-body-emphasis flex-fill">{name}</span>
        <span class="small fw-semibold text-uppercase text-nowrap text-{color}-emphasis">
            {$t(getStatusI18nKey(status))}
        </span>
    </div>
    {#if message}
        <p class="small mb-0">{message}</p>
    {/if}
    {#if startedAt}
        <p class="small text-body-secondary mb-0">
            {$t("checkers.dashboard.last-run-on", { date: formatCheckDate(startedAt) })}
        </p>
    {/if}
</a>
