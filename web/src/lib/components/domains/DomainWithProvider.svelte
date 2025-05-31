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
    import ImgProvider from "$lib/components/providers/ImgProvider.svelte";
    import type { Domain } from "$lib/model/domain";

    let className = '';
    export { className as class };

    export let domain: Domain;
</script>

<div
    class="d-flex my-1 {className}"
    style="min-width: 0"
>
    <div class="d-inline-block text-center" style="width: 50px;">
        <ImgProvider id_provider={domain.id_provider} />
    </div>
    <div class="font-monospace text-truncate flex-shrink-1">
        {domain.domain}
    </div>
</div>
