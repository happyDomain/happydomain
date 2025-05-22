<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2025 happyDomain
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
        Icon,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { getDomain as APIGetDomain } from "$lib/api/domains";
    import { controls as ctrlDiffZone } from "./ModalDiffZone.svelte";
    import { controls as ctrlDomainDelete } from "./ModalDomainDelete.svelte";
    import { diffZone as APIDiffZone } from "$lib/api/zone";
    import DiffSummary from "./DiffSummary.svelte";
    import type { Domain } from "$lib/model/domain";
    import { domains_idx } from "$lib/stores/domains";
    import { thisZone } from "$lib/stores/thiszone";
    import { t } from "$lib/translations";

    interface Props {
        domain: Domain;
        history: string;
    }

    let { domain, history }: Props = $props();

    async function getDomain(id: string): Promise<Domain> {
        return await APIGetDomain(id);
    }

    function showDiff(): void {
        if (!history) {
            return;
        }

        ctrlDiffZone.Open();
    }
</script>

{#if $domains_idx[domain.id] && $thisZone}
    {#if $domains_idx[domain.id].zone_history && history === $domains_idx[domain.id].zone_history[0]}
        {#key $thisZone}
            {#await APIDiffZone(domain, "@", $thisZone.id)}
                <Button
                    size="lg"
                    color="success"
                    outline
                    title={$t("domains.actions.propagate")}
                    on:click={showDiff}
                >
                    <Spinner size="sm" />
                    {$t("domains.actions.propagate")}
                </Button>
                <p class="mt-2 mb-1 text-center">
                    {$t("wait.wait")}
                </p>
            {:then zoneDiff}
                <Button
                    size="lg"
                    color="success"
                    outline={zoneDiff && !zoneDiff.length}
                    title={$t("domains.actions.propagate")}
                    on:click={showDiff}
                >
                    <Icon name="cloud-upload" aria-hidden="true" />
                    {$t("domains.actions.propagate")}
                </Button>
                <p class="mt-2 mb-1 text-center">
                    <DiffSummary {zoneDiff} />
                </p>
            {:catch err}
                <p class="mb-0 text-center">
                    <Icon name="exclamation-triangle" class="text-danger" />
                    {err}
                </p>
                <Button
                    color="danger"
                    class="mt-3"
                    outline
                    on:click={() => ctrlDomainDelete.Open()}
                >
                    <Icon name="trash" />
                    {$t("domains.stop")}
                </Button>
            {/await}
        {/key}
    {:else}
        <Button
            size="lg"
            color="warning"
            title={$t("domains.actions.rollback")}
            on:click={showDiff}
        >
            <Icon name="cloud-upload" aria-hidden="true" />
            {$t("domains.actions.rollback")}
        </Button>
        <p class="mt-2 mb-1 text-center">
            {#await getDomain(domain.id)}
                Chargement des informations de l'historique
            {:then domain}
                {#if domain.zone_meta && domain.zone_meta[history]}
                    {@const history_meta = domain.zone_meta[history]}
                    <span class="d-block text-truncate">
                        {#if history_meta.published}
                            Publiée le
                            {new Intl.DateTimeFormat(undefined, {
                                dateStyle: "long",
                                timeStyle: "long",
                            }).format(new Date(history_meta.published))}
                        {:else if history_meta.commit_date}
                            Enregistrée le
                            {new Intl.DateTimeFormat(undefined, {
                                dateStyle: "long",
                                timeStyle: "long",
                            }).format(new Date(history_meta.commit_date))}
                        {:else}
                            Dernière modification le
                            {new Intl.DateTimeFormat(undefined, {
                                dateStyle: "long",
                                timeStyle: "long",
                            }).format(new Date(history_meta.last_modified))}
                        {/if}
                    </span>
                    {#if history_meta.commit_message}
                        <span class="d-block text-truncate" title={history_meta.commit_message}>
                            {history_meta.commit_message}
                        </span>
                    {/if}
                {/if}
            {/await}
        </p>
    {/if}
{/if}
