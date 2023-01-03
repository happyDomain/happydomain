<script lang="ts">
 import {
     Input,
 } from 'sveltestrap';

 import { resolvers } from '$lib/resolver';
 import { t } from '$lib/translations';

 export let value = "local";
</script>

<Input
    type="select"
    bind:value={value}
    {...$$restProps}
>
    {#each Object.keys(resolvers) as resolver_kind}
        <optgroup label={resolver_kind}>
            {#each resolvers[resolver_kind] as resolver}
                <option value={resolver.value}>
                    {resolver.text}
                </option>
            {/each}
        </optgroup>
    {/each}
    <option value="custom">
        {$t('resolver.custom')}
    </option>
</Input>
