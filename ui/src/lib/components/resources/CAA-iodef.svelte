<script lang="ts">
 import { createEventDispatcher } from 'svelte';

 import {
     Button,
     Icon,
     Input,
 } from 'sveltestrap';

 const dispatch = createEventDispatcher();

 export let newone = false;
 export let readonly = false;
 export let value: any;

 let kind: string = "web";
 let url: string;

 $: if (value) switch (value.split(":")[0]) {
     case "mailto":
         kind = "mail";
         url = value.split(":")[1];
         break;
     default:
         kind = "web";
         url = value;
 }

 function updateValue(url) {
     if (kind == "mail") {
         value = "mailto:" + url;
     } else {
         value = url;
     }
 }

 $: updateValue(url);
</script>

<div class="d-flex gap-2 mb-2">
    <Input type="select" bind:value={kind}>
        <option value="mail">Mail</option>
        <option value="web">Webhook</option>
    </Input>

    <Input type={kind == "mail" ? "email" : "text"} bind:value={url} />

    {#if !newone}
        <Button
            type="button"
            color="danger"
            outline
            on:click={() => dispatch("delete-iodef")}
        >
            <Icon name="trash" />
        </Button>
    {:else}
        <Button
            color="success"
            outline
            type="button"
            disabled={!value}
            on:click={() => {dispatch("add-iodef", value); value = { }}}
        >
            <Icon name="plus" />
        </Button>
    {/if}
</div>
