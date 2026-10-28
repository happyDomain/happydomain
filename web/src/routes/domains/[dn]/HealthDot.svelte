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
    import { worstStatus, type CheckSection } from "$lib/domain_checks";
    import { t } from "$lib/translations";
    import { getStatusColor, getStatusI18nKey } from "$lib/utils/checkers";

    interface Props {
        // The checks of a subdomain; none draws an empty slot, keeping rows aligned.
        section?: CheckSection;
    }

    let { section }: Props = $props();

    let status = $derived(section ? worstStatus(section.items) : undefined);
</script>

<!-- Empty, its padding alone makes the dot. -->
{#if section}
    <span
        class="d-inline-block p-1 rounded-circle flex-shrink-0 bg-{getStatusColor(status)}"
        title={$t(getStatusI18nKey(status))}
    ></span>
{:else}
    <span class="d-inline-block p-1 flex-shrink-0"></span>
{/if}
