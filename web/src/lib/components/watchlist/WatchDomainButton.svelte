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
    import { resolve } from "$app/paths";
    import { Button, Spinner } from "@sveltestrap/sveltestrap";

    import { addDomainWatch } from "$lib/api/watchlist";
    import {
        ensureWatchlist,
        refreshWatchlist,
        upsertWatch,
        watchlist,
    } from "$lib/stores/watchlist";
    import { toasts } from "$lib/stores/toasts";
    import { userSession } from "$lib/stores/usersession";
    import { t } from "$lib/translations";
    import { findWatch, nameOf } from "$lib/utils/watchlist";

    interface Props {
        domain: string;
    }

    let { domain }: Props = $props();

    let pending = $state(false);

    $effect(() => {
        if ($userSession.id) {
            // Without the list, the button still offers to watch the domain.
            ensureWatchlist().catch(() => {});
        }
    });

    let watched = $derived(findWatch($watchlist, domain) !== undefined);

    async function watchIt() {
        pending = true;
        try {
            const w = await addDomainWatch({ domain, note: "" });
            if ($watchlist === undefined) await refreshWatchlist();
            else upsertWatch(w);
            toasts.addToast({
                title: $t("watchlist.added.title"),
                message: $t("watchlist.added.message", {
                    domain: nameOf(w) || domain,
                }),
                type: "success",
                timeout: 6000,
            });
        } catch (err) {
            toasts.addErrorToast({
                title: $t("watchlist.errors.add"),
                message: String(err),
                timeout: 8000,
            });
        } finally {
            pending = false;
        }
    }
</script>

{#if $userSession.id}
    {#if watched}
        <a href={resolve("/watchlist")} class="btn btn-sm btn-outline-success">
            <i class="bi bi-clock-history me-1"></i>
            {$t("watchlist.button.watched")}
        </a>
    {:else}
        <Button
            size="sm"
            color="primary"
            outline
            disabled={pending}
            onclick={watchIt}
            title={$t("watchlist.button.watch-hint")}
        >
            {#if pending}
                <Spinner size="sm" class="me-1" />
            {:else}
                <i class="bi bi-clock-history me-1"></i>
            {/if}
            {$t("watchlist.button.watch")}
        </Button>
    {/if}
{/if}
