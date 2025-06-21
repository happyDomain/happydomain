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
    import { Button, Icon } from "@sveltestrap/sveltestrap";

    import ProviderFormPage from "$lib/components/providers/FormPage.svelte";
    import type { Provider } from "$lib/model/provider";
    import type { ProviderSettingsState } from "$lib/model/provider_settings";
    import { t } from "$lib/translations";

    interface Props {
        data: { provider: Provider; provider_id: string };
    }

    let { data }: Props = $props();

    let value: ProviderSettingsState = $state({ ...data.provider, state: 0 });
</script>

<h1 class="text-center my-2">
    <Button type="button" class="fw-bolder" color="link" on:click={() => history.go(-1)}>
        <Icon name="chevron-left" />
    </Button>
    {$t("wait.updating")}
    {#if value}
        <em>{value._comment}</em>
    {/if}
</h1>
<hr class="mt-0 mb-0" />
<ProviderFormPage
    edit
    ptype={data.provider._srctype}
    state={0}
    providerId={data.provider_id}
    bind:value
/>
