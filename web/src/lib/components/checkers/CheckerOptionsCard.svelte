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
    import {
        Button,
        Card,
        CardBody,
        CardHeader,
        Form,
        FormGroup,
        Icon,
    } from "@sveltestrap/sveltestrap";

    import Resource from "$lib/components/inputs/Resource.svelte";
    import { t } from "$lib/translations";
    import { toasts } from "$lib/stores/toasts";

    interface Props {
        options: Array<any>;
        optionValues: Record<string, any>;
        title: string;
        saveOptionsFn: (values: Record<string, any>) => Promise<void | boolean>;
    }

    let { options, optionValues = $bindable(), title, saveOptionsFn }: Props = $props();

    let saving = $state(false);

    async function handleSave() {
        saving = true;
        try {
            await saveOptionsFn(optionValues);
            toasts.addToast({
                message: $t("checkers.messages.options-updated"),
                type: "success",
                timeout: 5000,
            });
        } catch (e: any) {
            toasts.addErrorToast({
                message: $t("checkers.messages.update-failed", { error: e.message }),
            });
        } finally {
            saving = false;
        }
    }
</script>

{#if options && options.length > 0}
    <Card class="mt-3">
        <CardHeader>
            <strong>{title}</strong>
        </CardHeader>
        <CardBody>
            <Form
                on:submit={(e) => {
                    e.preventDefault();
                    handleSave();
                }}
            >
                {#each options as optDoc}
                    {#if optDoc.id}
                        {@const optName = optDoc.id}
                        <FormGroup>
                            <Resource
                                edit={true}
                                index={optName}
                                specs={optDoc}
                                type={optDoc.type || "string"}
                                bind:value={optionValues[optName]}
                            />
                        </FormGroup>
                    {/if}
                {/each}
                <div class="d-flex gap-2">
                    <Button type="submit" color="success" disabled={saving}>
                        {#if saving}
                            <span class="spinner-border spinner-border-sm me-1"></span>
                        {/if}
                        <Icon name="check-circle"></Icon>
                        {$t("checkers.detail.save-changes")}
                    </Button>
                </div>
            </Form>
        </CardBody>
    </Card>
{/if}
