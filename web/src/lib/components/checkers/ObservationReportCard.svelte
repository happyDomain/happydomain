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
    import type { HappydnsObservationSnapshot } from "$lib/api-base/types.gen";

    interface Props {
        observations: HappydnsObservationSnapshot;
    }

    let { observations }: Props = $props();
</script>

{#if observations?.data && Object.keys(observations.data).length > 0}
    <div
        class="flex-fill"
        style="overflow: auto; padding: var(--bs-card-spacer-y) var(--bs-card-spacer-x)"
    >
        <pre class="mb-0" style="width: 0; min-width: 100%"><code
                >{JSON.stringify(observations.data, null, 2)}</code
            ></pre>
    </div>
{/if}
