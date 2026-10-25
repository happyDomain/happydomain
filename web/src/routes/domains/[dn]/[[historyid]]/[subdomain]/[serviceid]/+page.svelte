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
    import ServiceEditPage from "$lib/components/zones/ServiceEditPage.svelte";
    import type { Domain } from "$lib/model/domain";

    interface Props {
        data: {
            domain: Domain;
            history: string;
            zoneId: string;
            subdomain: string;
            serviceid: string;
        };
    }

    // The shape is what load() returns; a page reading part of it is not a
    // reason to describe it any less accurately.
    // eslint-disable-next-line svelte/no-unused-props
    let { data }: Props = $props();
</script>

<ServiceEditPage
    origin={data.domain}
    history={data.history}
    subdomain={data.subdomain}
    serviceid={data.serviceid}
/>
