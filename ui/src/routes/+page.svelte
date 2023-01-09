<script>
 import Home from '$lib/components/Home.svelte';
 import Onboarding from '$lib/components/Onboarding.svelte';
 import { domains } from '$lib/stores/domains';
</script>

{#if !$domains || $domains.length}
    <Home />
{:else}
    <Onboarding />
{/if}
