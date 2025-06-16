<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2025 happyDomain
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
    import { onMount, tick } from "svelte";

    import { Input } from "@sveltestrap/sveltestrap";

    import { filteredName } from "$lib/stores/serviceSelector";
    import { t } from "$lib/translations";

    onMount(async () => {
        await tick();

        const input = document.getElementById("filterserviceinput");
        if (input) {
            input.focus();
        }
    })
</script>

<Input
    id="filterserviceinput"
    autocomplete="off"
    placeholder={$t('common.filter')}
    bind:value={$filteredName}
    {...$$restProps}
/>
