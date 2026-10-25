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
    import { FormGroup, FormText, Input, Label } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";

    interface Props {
        // Prefixes the identifiers of the fields, unique in the page.
        id: string;
        name: string;
        description: string;
        origin?: string;
        // The origin is asked for a template made from a zone file.
        withOrigin?: boolean;
        disabled?: boolean;
    }

    let {
        id,
        name = $bindable(),
        description = $bindable(),
        origin = $bindable(""),
        withOrigin = false,
        disabled = false,
    }: Props = $props();
</script>

<FormGroup>
    <Label for="{id}-name">{$t("zone-templates.fields.name")}</Label>
    <Input
        id="{id}-name"
        required
        maxlength={100}
        placeholder={$t("zone-templates.fields.name-placeholder")}
        bind:value={name}
        {disabled}
    />
</FormGroup>
<FormGroup>
    <Label for="{id}-description">{$t("zone-templates.fields.description")}</Label>
    <Input
        id="{id}-description"
        type="textarea"
        rows={2}
        maxlength={1000}
        placeholder={$t("zone-templates.fields.description-placeholder")}
        bind:value={description}
        {disabled}
    />
</FormGroup>
{#if withOrigin}
    <FormGroup>
        <Label for="{id}-origin">{$t("zone-templates.fields.origin")}</Label>
        <Input
            id="{id}-origin"
            class="font-monospace"
            placeholder="example.com."
            autocapitalize="off"
            spellcheck="false"
            bind:value={origin}
            {disabled}
        />
        <FormText>{$t("zone-templates.fields.origin-help")}</FormText>
    </FormGroup>
{/if}
