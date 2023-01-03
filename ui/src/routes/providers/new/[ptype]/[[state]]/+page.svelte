<script lang="ts">
 import ProviderForm from '$lib/components/providers/Form.svelte';

 export let data: {ptype: string; state: number; providerId?: string;};
</script>

<ProviderForm
    ptype={data.ptype}
    state={data.state}
    providerId={data.providerId}
/>
