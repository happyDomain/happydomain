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
    import { createEventDispatcher } from "svelte";

    import { Icon } from "@sveltestrap/sveltestrap";

    import { nsrrtype } from "$lib/dns";
    import type { ServiceInfos } from "$lib/model/service_specs.svelte";
    import { serviceDescription, serviceName } from "$lib/services/infos";
    import { userSession } from "$lib/stores/usersession";

    const dispatch = createEventDispatcher();

    interface Props {
        active?: boolean;
        disabled?: boolean;
        reason?: string;
        svc: ServiceInfos;
    }

    let { active = false, disabled = false, reason = "", svc }: Props = $props();
</script>

<button
    type="button"
    class="list-group-item list-group-item-action d-flex gap-3 align-items-center px-3 py-2 border-0 rounded-3"
    class:active
    role="option"
    aria-selected={active}
    {disabled}
    onclick={() => dispatch("click")}
>
    {#if svc._svcicon}
        <img
            class="icon object-fit-contain"
            class:opacity-50={disabled}
            src={svc._svcicon}
            alt=""
        />
    {/if}
    <span class="d-grid flex-fill min-w-0">
        <span class="fw-semibold" class:opacity-50={disabled}>
            {$serviceName(svc)}
            {#if svc.record_types && $userSession.settings && $userSession.settings.showrrtypes}
                <span class="ms-1 small fw-medium text-body-secondary font-monospace">
                    {svc.record_types.map((rtype) => nsrrtype(rtype)).join(" ")}
                </span>
            {/if}
        </span>
        {#if reason}
            <span class="small fw-normal text-danger">{reason}</span>
        {:else}
            <span class="small fw-normal text-body-secondary">{$serviceDescription(svc)}</span>
        {/if}
    </span>
    <Icon name="check-lg" class="text-primary {active ? '' : 'invisible'}" />
</button>

<style>
    .icon {
        width: 2.25rem;
        max-height: 2.25rem;
    }
</style>
