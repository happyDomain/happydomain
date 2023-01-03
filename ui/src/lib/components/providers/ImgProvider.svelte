<script lang="ts">
 import { providers_idx } from '$lib/stores/providers';

 export let id_provider: string|undefined = undefined;
 export let ptype: string|undefined = undefined;

 export let style = "max-width: 100%; max-height: 2.5em; margin: -.6em .4em -.6em -.6em";
</script>

{#if ptype}
    <img
        src={'/api/providers/_specs/' + ptype + '/icon.png'}
        alt={ptype}
        title={ptype}
        {style}
        {...$$restProps}
    >
{:else if id_provider && $providers_idx && $providers_idx[id_provider]}
    <img
        src={'/api/providers/_specs/' + $providers_idx[id_provider]._srctype + '/icon.png'}
        alt={$providers_idx[id_provider]._srctype}
        title={$providers_idx[id_provider]._srctype}
        {style}
        {...$$restProps}
    >
{:else}
    <span {style} {...$$restProps}></span>
{/if}
