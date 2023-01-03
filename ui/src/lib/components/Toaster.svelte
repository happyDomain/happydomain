<script lang="ts">
 import {
     Toast,
     ToastBody,
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
            <ToastBody>
                {toast.message}
            </ToastBody>
        </Toast>
    {/each}
</div>
