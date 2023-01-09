<script lang="ts">
 import Home from '$lib/components/Home.svelte';
 import type { Provider } from '$lib/model/provider';

 export let data: {provider: Provider};
</script>

<Home filteredProvider={data.provider} />
