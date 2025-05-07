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
    // @ts-ignore
    import { escape } from "html-escaper";
    import { ModalHeader } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";

    export let toggle: () => void;
    export let dn: string;
    export let update = false;
</script>

<ModalHeader {toggle}>
    {#if update}
        {$t("service.update")}
    {:else}
        {@html $t("service.form-new", {
            domain: `<span class="font-monospace">${escape(dn)}</span>`,
        })}
    {/if}
</ModalHeader>
