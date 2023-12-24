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
     Toast,
     ToastHeader,
 } from 'sveltestrap';

 import { toasts } from '$lib/stores/toasts';
</script>

<div class="toast-container position-fixed top-0 end-0 p-3">
    {#each $toasts as toast}
        <Toast>
            <ToastHeader toggle={() => toast.dismiss()} icon={toast.getColor()}>
                {#if toast.title}{toast.title}{:else}happyDomain{/if}
            </ToastHeader>
            <div
                class="toast-body"
                style={toast.onclick?"cursor: pointer":""}
                on:click={() => { if (toast.onclick) toast.onclick(); }}
            >
                {toast.message}
            </div>
        </Toast>
    {/each}
</div>
