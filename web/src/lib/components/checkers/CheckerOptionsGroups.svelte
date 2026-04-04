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
    import { Badge, Card, CardBody, CardHeader } from "@sveltestrap/sveltestrap";
    import type { CheckerCheckerOptionDocumentation } from "$lib/api-base/types.gen";
    import { t } from "$lib/translations";

    let {
        groups,
    }: {
        groups: { key: string; label: string; opts: CheckerCheckerOptionDocumentation[] }[];
    } = $props();

    function autoFillLabel(key: string): string {
        const knownKeys: Record<string, string> = {
            domain_name: $t("checkers.auto-fill.domain_name"),
            subdomain: $t("checkers.auto-fill.subdomain"),
            service_type: $t("checkers.auto-fill.service_type"),
        };
        return knownKeys[key] || $t("checkers.auto-fill.generic", { key });
    }
</script>

{#each groups.filter((g) => g.opts.length > 0) as group}
    <Card class="mb-3">
        <CardHeader>
            <strong>{group.label}</strong>
            <Badge color="secondary" class="ms-2">{$t("checkers.detail.read-only")}</Badge>
        </CardHeader>
        <CardBody>
            <dl class="row mb-0">
                {#each group.opts as opt}
                    <dt class="col-sm-4">
                        {opt.label || opt.id}
                        {#if opt.autoFill}
                            <Badge color="info" class="ms-1">{autoFillLabel(opt.autoFill)}</Badge>
                        {/if}
                    </dt>
                    <dd class="col-sm-8">
                        <span class="text-muted small">{opt.type || "string"}</span>
                        {#if opt.description}
                            <div class="form-text">{opt.description}</div>
                        {/if}
                    </dd>
                {/each}
            </dl>
        </CardBody>
    </Card>
{/each}
