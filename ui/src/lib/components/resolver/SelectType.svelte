<script lang="ts">
 import {
     Input,
 } from 'sveltestrap';

 import { dns_common_types } from '$lib/dns';

 export let value = "ANY";
</script>

<Input
    type="select"
    bind:value={value}
    {...$$restProps}
>
    {#each dns_common_types as option}
        <option>{option}</option>
    {/each}
</Input>
