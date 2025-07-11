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
    import { createEventDispatcher } from "svelte";

    import { Col, Row } from "@sveltestrap/sveltestrap";

    import type { Field } from "$lib/model/custom_form";
    import ResourceRawInput from "./raw.svelte";

    const dispatch = createEventDispatcher();

    interface Props {
        alwaysShow?: boolean;
        edit?: boolean;
        index: string;
        showDescription?: boolean;
        specs: Field;
        value: any;
        [key: string]: any
    }

    let {
        alwaysShow = false,
        edit = false,
        index,
        showDescription = true,
        specs,
        value = $bindable(),
        ...rest
    }: Props = $props();
</script>

{#if alwaysShow || edit || value != null}
    <Row {...rest}>
        <label
            for={"spec-" + index + "-" + specs.id}
            title={specs.label}
            class="col-md-4 col-form-label text-truncate text-md-right text-primary"
        >
            {#if specs.label}
                {specs.label}
            {:else}
                {specs.id}
            {/if}
        </label>
        <Col md="8" class="d-flex flex-column">
            <div class="flex-fill d-flex align-items-center">
                <ResourceRawInput
                    {edit}
                    {index}
                    {specs}
                    bind:value
                    on:focus={() => dispatch("focus")}
                    on:blur={() => dispatch("blur")}
                />
            </div>
            {#if specs.description && (showDescription || (specs.choices && specs.choices.length > 0))}
                <p class="text-justify" style="line-height: 1.1">
                    <small class="text-muted">{specs.description}</small>
                </p>
            {/if}
        </Col>
    </Row>
{/if}
