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
    import { Badge, Card, CardBody, CardHeader, Table } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";
    import type { HappydnsCheckEvaluation } from "$lib/api-base/types.gen";
    import { getStatusColor, getStatusI18nKey } from "$lib/utils";

    interface Props {
        evaluation: HappydnsCheckEvaluation;
    }

    let { evaluation }: Props = $props();
</script>

<Card>
    <CardHeader>
        <strong>{$t("checkers.detail.check-rules")}</strong>
    </CardHeader>
    <CardBody>
        {#if evaluation.states && evaluation.states.length > 0}
            <Table class="mb-0" size="sm" borderless hover>
                <thead>
                    <tr>
                        <th>{$t("checkers.result.field.rule")}</th>
                        <th>{$t("checkers.result.field.status")}</th>
                        <th>{$t("checkers.result.field.message")}</th>
                    </tr>
                </thead>
                <tbody>
                    {#each evaluation.states as state}
                        <tr>
                            <td>
                                <code>{state.rule ?? ""}</code>
                                {#if state.code}<small class="text-muted"> · {state.code}</small>{/if}
                            </td>
                            <td><Badge color={getStatusColor(state.status)}>{$t(getStatusI18nKey(state.status))}</Badge></td>
                            <td>
                                {#if state.subject}<strong>{state.subject}</strong>{#if state.message}: {/if}{/if}{state.message ?? ""}
                            </td>
                        </tr>
                    {/each}
                </tbody>
            </Table>
        {:else}
            <pre class="mb-0"><code>{JSON.stringify(evaluation, null, 2)}</code></pre>
        {/if}
    </CardBody>
</Card>
