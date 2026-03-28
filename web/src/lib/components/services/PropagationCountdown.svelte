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
    import { onDestroy } from "svelte";

    import { formatCountdown } from "$lib/utils/datetime";
    import { t } from "$lib/translations";

    interface Props {
        isPropagating?: boolean;
        localeString?: string;
        propagatedAt?: string | null;
    }

    let {
        isPropagating = $bindable(false),
        propagatedAt,
        localeString = "service.propagation-remaining",
    }: Props = $props();

    let _propagatedAt = $derived(propagatedAt ? new Date(propagatedAt) : null);

    let countdown = $state("");
    let interval: ReturnType<typeof setInterval>;

    onDestroy(() => {
        if (interval) clearInterval(interval);
    });

    $effect(() => {
        if (_propagatedAt) {
            isPropagating = _propagatedAt > new Date();

            if (interval) clearInterval(interval);

            countdown = formatCountdown(_propagatedAt);
            interval = setInterval(() => {
                countdown = formatCountdown(_propagatedAt);
                if (_propagatedAt <= new Date()) {
                    isPropagating = false;
                    clearInterval(interval);
                }
            }, 1000);
        } else if (interval) {
            clearInterval(interval);
        }
    });
</script>

{#if propagatedAt}
    <span style="font-variant-numeric: tabular-nums">{$t(localeString, { countdown })}</span>
{/if}
