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
    import type { ClassValue } from "svelte/elements";

    import { Input } from "@sveltestrap/sveltestrap";

    import { isGlobalTemplate, type ZoneTemplate } from "$lib/model/zone_template";
    import { t } from "$lib/translations";

    interface Props {
        class?: ClassValue;
        // The templates to choose from; the current one is shown alone until
        // they are loaded.
        templates: ZoneTemplate[] | undefined;
        current: ZoneTemplate;
        onselect: (id: string) => void;
    }

    let { class: className, templates, current, onselect }: Props = $props();

    let list = $derived(
        templates?.some((tpl) => tpl.id === current.id)
            ? templates
            : [current, ...(templates ?? [])],
    );
    let mine = $derived(list.filter((tpl) => !isGlobalTemplate(tpl)));
    let globals = $derived(list.filter((tpl) => isGlobalTemplate(tpl)));
</script>

<Input
    type="select"
    class={className}
    aria-label={$t("zone-templates.title")}
    value={current.id}
    on:change={(e: Event) => onselect((e.target as HTMLSelectElement).value)}
>
    {#if mine.length > 0}
        <optgroup label={$t("zone-templates.mine")}>
            {#each mine as tpl (tpl.id)}
                <option value={tpl.id}>{tpl.name}</option>
            {/each}
        </optgroup>
    {/if}
    {#if globals.length > 0}
        <optgroup label={$t("zone-templates.global")}>
            {#each globals as tpl (tpl.id)}
                <option value={tpl.id}>{tpl.name}</option>
            {/each}
        </optgroup>
    {/if}
</Input>
