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
    import { goto } from "$app/navigation";
    import { templateLinks } from "$links";

    import { Badge, Button, ButtonGroup, Icon, Table } from "@sveltestrap/sveltestrap";

    import { countServices, isGlobalTemplate, type ZoneTemplate } from "$lib/model/zone_template";
    import { locale, t } from "$lib/translations";
    import { formatRelativeTime } from "$lib/utils/datetime";

    interface Props {
        items: Array<ZoneTemplate>;
        onedit: (template: ZoneTemplate) => void;
        ondelete: (template: ZoneTemplate) => void;
    }

    let { items, onedit, ondelete }: Props = $props();

    function open(template: ZoneTemplate) {
        goto(templateLinks().template(encodeURIComponent(template.id)));
    }
</script>

<Table striped hover responsive>
    <thead>
        <tr>
            <th>{$t("zone-templates.fields.name")}</th>
            <th>{$t("zone-templates.fields.description")}</th>
            <th>{$t("zone-templates.services-title")}</th>
            <th>{$t("zone-templates.last-modified")}</th>
            <th></th>
        </tr>
    </thead>
    <tbody>
        {#each items as template (template.id)}
            <tr style="cursor: pointer" onclick={() => open(template)}>
                <td class="text-nowrap">
                    <a
                        href={templateLinks().template(encodeURIComponent(template.id))}
                        class="fw-bold text-reset text-decoration-none"
                        onclick={(e) => e.stopPropagation()}>{template.name}</a
                    >
                    {#if isGlobalTemplate(template)}
                        <Badge color="info" class="ms-2" title={$t("zone-templates.global-hint")}>
                            <i class="bi bi-globe me-1"></i>{$t("zone-templates.global")}
                        </Badge>
                    {/if}
                </td>
                <td class="text-muted description">{template.description ?? ""}</td>
                <td class="text-nowrap">
                    {$t("zone-templates.services", { count: countServices(template) })}
                </td>
                <td class="text-nowrap text-muted small">
                    {#if template.last_modified}
                        <span title={new Date(template.last_modified).toLocaleString($locale)}>
                            {formatRelativeTime(template.last_modified, $locale)}
                        </span>
                    {/if}
                </td>
                <td class="text-end">
                    <ButtonGroup size="sm">
                        <Button
                            color="outline-secondary"
                            title={$t("zone-templates.actions.view")}
                            onclick={(e) => {
                                e.stopPropagation();
                                open(template);
                            }}
                        >
                            <Icon name="eye" />
                        </Button>
                        {#if !isGlobalTemplate(template)}
                            <Button
                                color="outline-secondary"
                                title={$t("zone-templates.edit")}
                                onclick={(e) => {
                                    e.stopPropagation();
                                    onedit(template);
                                }}
                            >
                                <Icon name="pencil" />
                            </Button>
                            <Button
                                color="outline-danger"
                                title={$t("zone-templates.delete")}
                                onclick={(e) => {
                                    e.stopPropagation();
                                    ondelete(template);
                                }}
                            >
                                <Icon name="trash" />
                            </Button>
                        {/if}
                    </ButtonGroup>
                </td>
            </tr>
        {/each}
    </tbody>
</Table>

<style>
    .description {
        max-width: 24rem;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
</style>
