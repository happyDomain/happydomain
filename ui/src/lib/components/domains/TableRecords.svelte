<script lang="ts">
 import {
     Badge,
     Card,
     CardBody,
     CardText,
     CardTitle,
     CardSubtitle,
     Icon,
     ListGroup,
     ListGroupItem,
     Table,
     Spinner,
 } from 'sveltestrap';

 import Record from '$lib/components/domains/Record.svelte';

 export let serviceRecords: Array<ServiceRecord>|null = null;
</script>

{#if serviceRecords}
    <Table
        class="m-0"
        hover
        size="sm"
        striped
    >
        <tbody>
            {#each serviceRecords as record}
                <Record {record} />
            {/each}
        </tbody>
    </Table>
{/if}
