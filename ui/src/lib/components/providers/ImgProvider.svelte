<script lang="ts">
 export let id_provider;
</script>

{#if providers_getAll[id_provider]}
    <img
        src="'/api/providers/_specs/' + providers_getAll[id_provider]._srctype + '/icon.png'"
        alt={providers_getAll[id_provider]._srctype}
        title={providers_getAll[id_provider]._srctype}
        style="max-width: 100%; max-height: 2.5em; margin: -.6em .4em -.6em -.6em"
    >
{/if}
