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
        Badge,
        Card,
        CardBody,
        CardHeader,
        ListGroup,
        ListGroupItem,
    } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import type {
        HappydnsCheckEvaluation,
        HappydnsCheckState,
        HappydnsStatus,
    } from "$lib/api-base/types.gen";
    import { getStatusColor, getStatusI18nKey } from "$lib/utils";

    interface Props {
        evaluation: HappydnsCheckEvaluation;
    }

    let { evaluation }: Props = $props();

    const groups = $derived.by(() => {
        const map = new Map<string, HappydnsCheckState[]>();
        for (const s of evaluation.states ?? []) {
            const key = s.rule ?? "";
            const existing = map.get(key);
            if (existing) existing.push(s);
            else map.set(key, [s]);
        }
        return Array.from(map.entries()).map(([rule, states]) => ({
            rule,
            states,
            worst: states.reduce<HappydnsStatus | undefined>((acc, s) => {
                if (s.status == null) return acc;
                return acc === undefined || s.status > acc ? s.status : acc;
            }, undefined),
        }));
    });
</script>

{#if groups.length > 0}
    {#each groups as group}
        <Card class="mb-3">
            <CardHeader class="d-flex justify-content-between align-items-center">
                <code>{group.rule}</code>
                <Badge color={getStatusColor(group.worst)}>
                    {$t(getStatusI18nKey(group.worst))}
                </Badge>
            </CardHeader>
            <ListGroup flush>
                {#each group.states as state}
                    <ListGroupItem>
                        <div class="d-flex justify-content-between align-items-start gap-3">
                            <div>
                                {#if state.subject}
                                    <div><strong>{state.subject}</strong></div>
                                {/if}
                                {#if state.code}
                                    <small class="text-muted">{state.code}</small>
                                {/if}
                                {#if state.message}
                                    <div>{state.message}</div>
                                {/if}
                                {#if state.meta && typeof state.meta.hint === "string"}
                                    <div class="small text-muted mt-2">{state.meta.hint}</div>
                                {/if}
                            </div>
                            <Badge color={getStatusColor(state.status)}>
                                {$t(getStatusI18nKey(state.status))}
                            </Badge>
                        </div>
                    </ListGroupItem>
                {/each}
            </ListGroup>
        </Card>
    {/each}
{:else}
    <Card>
        <CardHeader>
            <strong>{$t("checkers.detail.check-rules")}</strong>
        </CardHeader>
        <CardBody>
            <pre class="mb-0"><code>{JSON.stringify(evaluation, null, 2)}</code></pre>
        </CardBody>
    </Card>
{/if}
