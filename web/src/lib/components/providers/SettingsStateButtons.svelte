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

    import { Button, Spinner } from "@sveltestrap/sveltestrap";

    import type { CustomForm } from "$lib/model/custom_form.svelte";
    import { t } from "$lib/translations";

    const dispatch = createEventDispatcher();

    interface Props {
        canDoNext?: boolean;
        edit?: boolean;
        form?: CustomForm | null;
        nextInProgress?: boolean;
        previousInProgress?: boolean;
        submitForm?: string | null;
        [key: string]: any
    }

    let {
        canDoNext = true,
        edit = false,
        form = null,
        nextInProgress = false,
        previousInProgress = false,
        submitForm = null,
        ...rest
    }: Props = $props();

    let disabled = $derived(nextInProgress || previousInProgress);
</script>

<div {...rest}>
    {#if form}
        {#if (!form.previousEditButtonText || !edit) && form.previousButtonText}
            <Button
                type="button"
                class="mx-1"
                color="secondary"
                outline
                {disabled}
                on:click={() => dispatch("previous-state")}
            >
                {#if previousInProgress}
                    <Spinner size="sm" />
                {/if}
                {$t(form.previousButtonText)}
            </Button>
        {/if}
        {#if (!form.nextEditButtonText || !edit) && form.nextButtonText}
            <Button
                type="submit"
                class="mx-1"
                color="primary"
                disabled={disabled || !canDoNext}
                form={submitForm}
            >
                {#if nextInProgress}
                    <Spinner size="sm" />
                {/if}
                {$t(form.nextButtonText)}
            </Button>
        {/if}
        {#if edit && form.previousEditButtonText}
            <Button
                type="button"
                class="mx-1"
                color="secondary"
                outline
                {disabled}
                on:click={() => dispatch("previous-state")}
            >
                {#if previousInProgress}
                    <Spinner size="sm" />
                {/if}
                {$t(form.previousEditButtonText)}
            </Button>
        {/if}
        {#if edit && form.nextEditButtonText}
            <Button
                type="submit"
                class="mx-1"
                color="primary"
                disabled={disabled || !canDoNext}
                form={submitForm}
            >
                {#if nextInProgress}
                    <Spinner size="sm" />
                {/if}
                {$t(form.nextEditButtonText)}
            </Button>
        {/if}
    {:else}
        <Button
            type="button"
            class="mx-1"
            color="secondary"
            outline
            {disabled}
            on:click={() => dispatch("previous-state")}
        >
            {#if previousInProgress}
                <Spinner size="sm" />
            {/if}
            {$t("common.cancel")}
        </Button>
        <Button
            type="submit"
            class="mx-1"
            color="primary"
            disabled={disabled || !canDoNext}
            form={submitForm}
        >
            {#if nextInProgress}
                <Spinner size="sm" />
            {/if}
            {$t("common.next")} &gt;
        </Button>
    {/if}
</div>
