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
    import PageTitle from "$lib/components/PageTitle.svelte";
    import type { Domain } from "$lib/model/domain";
    import { t } from "$lib/translations";

    interface Props {
        data: { domain: Domain };
    }

    // Reached only without any version: the load redirects to the latest one.
    let { data }: Props = $props();
</script>

<div class="flex-fill pb-4 pt-2">
    <PageTitle
        title={$t("history.title")}
        domain={data.domain.domain}
        subtitle={$t("history.subtitle")}
    />
    <p class="text-center text-muted mt-5">{$t("history.empty")}</p>
</div>
