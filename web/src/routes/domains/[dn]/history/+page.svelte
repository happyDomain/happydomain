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
    import { Accordion, AccordionItem, Button, Icon, Spinner } from "@sveltestrap/sveltestrap";

    import { getDomain } from "$lib/api/domains";
    import DiffZone from "$lib/components/zones/DiffZone.svelte";
    import type { Domain, ZoneHistory } from "$lib/model/domain";
    import { getUser } from "$lib/stores/users";
    import { t } from "$lib/translations";

    interface Props {
        data: { domain: Domain; history: string };
    }

    let { data }: Props = $props();

    let isOpen: Record<string, boolean> = $state({});
    if (data.domain.zone_history && data.domain.zone_history.length > 0) {
        isOpen[data.domain.zone_history[0]] = true;
    }

    function isSameMonth(a: Date, b: Date): boolean {
        return a.getMonth() === b.getMonth();
    }
</script>

<div class="flex-fill pb-4 pt-2">
    <h2>{$t("history.title")} <span class="font-monospace">{data.domain.domain}</span></h2>
    {#await getDomain(data.domain.id)}
        <div class="mt-5 text-center flex-fill">
            <Spinner />
            <p>{$t("wait.loading")}</p>
        </div>
    {:then domain}
      {#if domain.zone_history && domain.zone_meta}
        {#each domain.zone_history as zid, idx}
            {@const history = domain.zone_meta[zid]}
            {@const moddate = new Intl.DateTimeFormat(undefined, {
                dateStyle: "long",
                timeStyle: "medium",
            }).format(new Date(history.last_modified))}
            {#if idx === 0 || !isSameMonth(new Date(domain.zone_meta[domain.zone_history[idx - 1]].last_modified), new Date(history.last_modified))}
                <h3 class="mt-4 fw-bolder">
                    <Icon name="calendar2-month" />
                    {new Intl.DateTimeFormat(undefined, { month: "long", year: "numeric" }).format(
                        new Date(history.last_modified),
                    )}
                </h3>
            {/if}
            <h4 class="mt-4 d-flex gap-2 align-items-center">
                {#await getUser(history.id_author)}
                    <img
                        src={"/api/users/" + encodeURIComponent(history.id_author) + "/avatar.png"}
                        alt={history.id_author}
                        style="height: 1.1em; border-radius: .1em"
                    />{moddate}
                {:then user}
                    <img
                        src={"/api/users/" + encodeURIComponent(history.id_author) + "/avatar.png"}
                        alt={user.email}
                        title={user.email}
                        style="height: 1.1em; border-radius: .1em"
                    />{moddate} <span class="text-muted">{$t("by")} {user.email}</span>
                {:catch}
                    {moddate}
                {/await}
                <Button
                    color="primary"
                    outline
                    href={"/domains/" + encodeURIComponent(data.domain.id) + "/" + history.id}
                    size="sm"
                    title={$t("history.see")}
                >
                    <Icon name="eye-fill" />
                </Button>
            </h4>
            <div class="row row-cols-3 text-center">
                {#if history.published}
                    <div class="col">
                        {$t("history.published-on")}<br />
                        <strong>
                            {new Intl.DateTimeFormat(undefined, { dateStyle: "long" }).format(
                                new Date(history.published),
                            )}<br />
                            {new Intl.DateTimeFormat(undefined, { timeStyle: "medium" }).format(
                                new Date(history.published),
                            )}
                        </strong>
                    </div>
                {/if}
                {#if history.commit_date}
                    <div class="col">
                        {$t("history.committed-on")}<br />
                        {new Intl.DateTimeFormat(undefined, { dateStyle: "long" }).format(
                            new Date(history.commit_date),
                        )}<br />
                        {new Intl.DateTimeFormat(undefined, { timeStyle: "medium" }).format(
                            new Date(history.commit_date),
                        )}
                    </div>
                {/if}
                {#if history.last_modified}
                    <div class="col">
                        {$t("history.modified-on")}<br />
                        {new Intl.DateTimeFormat(undefined, { dateStyle: "long" }).format(
                            new Date(history.last_modified),
                        )}<br />
                        {new Intl.DateTimeFormat(undefined, { timeStyle: "medium" }).format(
                            new Date(history.last_modified),
                        )}
                    </div>
                {/if}
            </div>
            {#if history.commit_message}
                <p class="mb-1">
                    {history.commit_message}
                </p>
            {/if}

            {#if idx < domain.zone_history.length - 1}
                <Accordion class="mt-3">
                    <AccordionItem
                        active={isOpen[history.id]}
                        header={$t("history.diff")}
                        on:toggle={(evt) => {
                            isOpen[history.id] = evt.detail;
                        }}
                    >
                        {#if isOpen[history.id]}
                            <DiffZone
                                {domain}
                                zoneFrom={history.id}
                                zoneTo={domain.zone_history[idx + 1]}
                            />
                        {/if}
                    </AccordionItem>
                </Accordion>
            {/if}
        {/each}
      {:else}
        No history yet.
      {/if}
    {/await}
</div>
