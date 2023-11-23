<script lang="ts">
 import {
     Button,
     Icon,
     Spinner,
 } from 'sveltestrap';

 import { getDomain } from '$lib/api/domains';
 import { t } from '$lib/translations';

 export let data: {domain: DomainInList; history: string; streamed: Object;};
</script>

<div class="flex-fill pb-4 pt-2">
    <h2>Historique des changements la zone <span class="font-monospace">{data.domain.domain}</span></h2>
    {#await getDomain(data.domain.id)}
        <div class="mt-5 text-center flex-fill">
            <Spinner label="Spinning" />
            <p>{$t('wait.loading')}</p>
        </div>
    {:then domain}
        {#each domain.zone_history as history}
            <h3 class="mt-3">
                {new Intl.DateTimeFormat(undefined, {dateStyle: "long", timeStyle: "long"}).format(new Date(history.last_modified))}
                <small class="text-muted">
                    par {history.id_author}
                </small>
                <Button
                    color="primary"
                    href={"/domains/" + encodeURIComponent(data.domain.domain) + "/" + history.id}
                    size="sm"
                    title="Voir la zone"
                >
                    <Icon name="arrow-right" />
                </Button>
            </h3>
            {#if history.published}
                <p class="mb-1">
                    <strong>Publiée le
                        {new Intl.DateTimeFormat(undefined, {dateStyle: "long", timeStyle: "long"}).format(new Date(history.published))}
                    </strong>
                </p>
            {/if}
            {#if history.commit_date}
                <p class="mb-1">
                    Enregistrée le
                    {new Intl.DateTimeFormat(undefined, {dateStyle: "long", timeStyle: "long"}).format(new Date(history.commit_date))}
                </p>
            {/if}
            {#if history.last_modified}
                <p class="mb-1">
                    Dernière modification le
                    {new Intl.DateTimeFormat(undefined, {dateStyle: "long", timeStyle: "long"}).format(new Date(history.last_modified))}
                </p>
            {/if}
            {#if history.commit_message}
                <p class="mb-1">
                    {history.commit_message}
                </p>
            {/if}
        {/each}
    {/await}
</div>
