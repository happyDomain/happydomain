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
