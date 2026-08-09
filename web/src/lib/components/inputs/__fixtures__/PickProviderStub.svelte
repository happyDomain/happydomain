<!--
     Test-only stand-in for $lib/components/modals/PickProvider.svelte.

     The real modal only calls its `ondone` prop once a user has clicked
     through a multi-step provider picker inside a Bootstrap modal, which
     isn't worth driving through the DOM just to exercise NewDomain.svelte's
     onProviderSelected callback. This stub exposes that same callback
     behind a plain button instead.
-->
<script lang="ts">
    import type { Provider } from "$lib/model/provider";

    interface Props {
        ondone?: (provider: Provider) => void;
    }

    let { ondone }: Props = $props();
</script>

<button type="button" onclick={() => ondone?.({ _id: "picked-provider" } as Provider)}>
    pick provider
</button>
