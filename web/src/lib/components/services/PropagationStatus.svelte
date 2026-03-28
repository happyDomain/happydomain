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

    import { t } from "$lib/translations";
    import PropagationCountdown from "./PropagationCountdown.svelte";

    interface Props {
        propagatedAt?: string | null;
    }

    let { propagatedAt }: Props = $props();

    let _propagatedAt = $derived(propagatedAt ? new Date(propagatedAt) : null);

    let isPropagating = $derived(_propagatedAt && _propagatedAt > new Date());
</script>

{#if isPropagating}
    <div class="border rounded p-2 mt-3 bg-white">
        <div class="d-flex align-items-center gap-2 mb-1">
            <Icon name="broadcast-pin" class="text-warning" />
            <strong class="small">{$t("service.propagation-in-progress")}</strong>
        </div>
        <p class="small text-muted mb-2">
            <PropagationCountdown bind:isPropagating {propagatedAt} />
        </p>
    </div>
{/if}
