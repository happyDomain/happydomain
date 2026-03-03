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

<script module lang="ts">
    export const controls = {
        Open (domain: string): void { },
    };
</script>

<script lang="ts">
    import ServiceSelectorModal, {
        controls as ctrlServiceSelector,
    } from "$lib/components/modals/ServiceSelector.svelte";
    import type { Domain } from "$lib/model/domain";
    import type { Zone } from "$lib/model/zone";
    import { navigate } from "$lib/stores/config";

    interface Props {
        origin: Domain;
        zone: Zone;
        historyId: string;
    }

    let { origin, zone, historyId }: Props = $props();

    function Open(domain: string): void {
        ctrlServiceSelector.Open(domain);
    }

    controls.Open = Open;

    function onShowNextModal(event: CustomEvent<{ _svctype: string; _domain: string }>) {
        const { _svctype, _domain } = event.detail;
        const subdomainParam = _domain === "" ? "@" : _domain;
        navigate(
            `/domains/${encodeURIComponent(origin.domain)}/${encodeURIComponent(historyId)}/${encodeURIComponent(subdomainParam)}/new?type=${encodeURIComponent(_svctype)}`,
        );
    }
</script>

<ServiceSelectorModal
    {origin}
    zservices={zone.services}
    on:show-next-modal={onShowNextModal}
/>
