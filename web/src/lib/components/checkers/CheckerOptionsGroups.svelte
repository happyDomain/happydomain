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
    import { Card, CardBody, CardHeader } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";

    const AUTO_FILL_KEYS: Record<string, string> = {
        domain_name: "checks.auto-fill.domain_name",
        subdomain: "checks.auto-fill.subdomain",
        service_type: "checks.auto-fill.service_type",
    };

    function getAutoFillLabel(autoFill: string): string {
        const tKey = AUTO_FILL_KEYS[autoFill];
        if (tKey) return $t(tKey);
        return $t("checks.auto-fill.generic", { key: autoFill });
    }

    interface OptionDef {
        id?: string;
        label?: string;
        type?: string;
        default?: unknown;
        placeholder?: string;
        description?: string;
        required?: boolean;
        autoFill?: string;
    }

    interface OptionGroup {
        label: string;
        opts: OptionDef[];
    }

    interface Props {
        groups: OptionGroup[];
    }

    let { groups }: Props = $props();
</script>

{#each groups as optGroup}
    {#if optGroup.opts.length > 0}
        <Card class="mb-3">
            <CardHeader>
                <strong>{optGroup.label}</strong>
                <small class="text-muted ms-2">{$t("checks.detail.read-only")}</small>
            </CardHeader>
            <CardBody>
                <dl class="row mb-0">
                    {#each optGroup.opts as optDoc}
                        {@const optName = optDoc.id!}
                        <dt class="col-sm-4">
                            {optDoc.label || optDoc.id}:
                        </dt>
                        <dd class="col-sm-8">
                            {#if optDoc.autoFill}
                                <span class="badge bg-info me-1">
                                    {getAutoFillLabel(optDoc.autoFill)}
                                </span>
                            {:else if optDoc.default}
                                <span class="text-muted d-block">{optDoc.default}</span>
                            {:else if optDoc.placeholder}
                                <em class="text-muted d-block">{optDoc.placeholder}</em>
                            {/if}
                            {#if optDoc.description}
                                <small class="text-muted d-block">{optDoc.description}</small>
                            {/if}
                            <small class="text-muted">
                                {$t("checks.option-groups.type", {
                                    type: optDoc.type || "string",
                                })}
                            </small>
                            {#if optDoc.required && !optDoc.autoFill}
                                <small class="text-danger ms-2">
                                    {$t("checks.option-groups.required")}
                                </small>
                            {/if}
                        </dd>
                    {/each}
                </dl>
            </CardBody>
        </Card>
    {/if}
{/each}
