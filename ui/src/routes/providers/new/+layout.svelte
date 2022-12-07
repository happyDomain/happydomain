<script lang="ts">
 import {
     Button,
     Container,
     Icon,
 } from 'sveltestrap';

 import { t } from '$lib/translations';
</script>

<Container class="d-flex flex-column mt-4" fluid>
    <h1 class="text-center mb-3">
        <Button
            type="button"
            class="fw-bolder"
            color="link"
            on:click={() => history.go(-1)}
        >
            <Icon name="chevron-left" />
        </Button>
        {$t("provider.select-provider")}
    </h1>
    <hr class="mt-0 mb-0">
    <slot class="flex-grow-1"></slot>
</Container>
