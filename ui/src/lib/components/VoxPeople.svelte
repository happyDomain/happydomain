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
 import {
     Icon,
 } from 'sveltestrap';

 const { MODE } = import.meta.env;

 import { userSession } from '$lib/stores/usersession';
 import { t, locale } from '$lib/translations';

 export let routeId: string | null;
 const instancename = encodeURIComponent(window.location.hostname);
</script>

{#if MODE == 'production'}
    <a
        id="voxpeople"
        href="https://framaforms.org/quel-est-votre-avis-sur-happydns-1610366701?u={$userSession?$userSession.id:0}&amp;i={instancename}&amp;p={routeId}&amp;l={$locale}"
        target="_blank"
        rel="noreferrer"
        title={$t('common.survey')}
        class="d-flex btn btn-light justify-content-center align-items-center"
        data-umami-event="vox-people"
    >
        <Icon
            name="chat-right-text"
        />
    </a>
{/if}

<style>
 #voxpeople {
     position: fixed;
     bottom: 7vh;
     right: 4vw;
     z-index: 1051;
     height: max(4vw, 4vh);
     width: max(4vw, 4vh);
     border-radius: 4vw;
     box-shadow: 0 0px 3px 0 #9332bb;
     font-size: max(1.7vw, 1.7vh);
 }
</style>
