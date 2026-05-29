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
    import type { ClassValue } from "svelte/elements";
    import { goto } from "$app/navigation";

    import ImgDomain from "$lib/components/domains/ImgDomain.svelte";
    import ImgProvider from "$lib/components/providers/ImgProvider.svelte";
    import type { HappydnsDomain } from "$lib/api-base/types.gen";

    interface Props {
        class?: ClassValue;
        domain: Pick<HappydnsDomain, "domain" | "id_provider">;
    }

    let { class: className, domain }: Props = $props();
</script>

<div class={["d-flex align-items-center", className]} style="min-width: 0">
    <div class="icons position-relative flex-shrink-0 me-2">
        <ImgDomain
            domain={domain.domain}
            style="width: inherit; height: inherit; object-fit: scale-down;"
        />
        {#if domain.id_provider}
            <button
                type="button"
                class="provider-badge"
                onclick={(event) => {
                    event.preventDefault();
                    event.stopPropagation();
                    goto(`/?provider=${encodeURIComponent(domain.id_provider)}`);
                }}
            >
                <ImgProvider
                    id_provider={domain.id_provider}
                    style="display: block; width: 100%; height: 100%; object-fit: contain; border-radius: 50%;"
                />
            </button>
        {/if}
    </div>
    <div class="font-monospace text-truncate flex-shrink-1">
        {domain.domain}
    </div>
</div>

<style>
    .icons {
        width: 2em;
        height: 2em;
        line-height: 0;
    }

    /*
       The provider icon sits on artwork of any colour, and most of them are
       drawn for a white page. Give it its own white disc, DuckDuckGo style:
       the padding is the white ring that keeps a dark logo legible, and the
       overflow crops whatever the provider ships to that same disc.
    */
    .provider-badge {
        border: none;
        text-decoration: none;
        appearance: none;
        margin: 0;
        position: absolute;
        right: -7px;
        bottom: -5px;
        box-sizing: border-box;
        display: block;
        width: 20px;
        height: 20px;
        padding: 1.5px;
        overflow: hidden;
        background: #fff;
        border-radius: 50%;
        box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.08);
        cursor: pointer;
    }
</style>
