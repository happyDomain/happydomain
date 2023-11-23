<script lang="ts">
 import { goto } from '$app/navigation';

 import {
     Alert,
     Icon,
     Spinner,
 } from 'sveltestrap';

 import type { DomainInList } from '$lib/model/domain';
 import { retrieveZone } from '$lib/stores/thiszone';
 import { t } from '$lib/translations';

 export let data: {domain: DomainInList;};

 let rz = retrieveZone(data.domain);
 rz.then(() => {
     goto(`/domains/${encodeURIComponent(data.domain.domain)}`);
 }, (e) => { })
</script>

<div class="mt-4 text-center flex-fill">
    {#await rz}
        <Spinner label={$t('common.spinning')} />
        <p>{$t('wait.importing')}</p>
    {:then}
        <p>{$t('wait.wait')}</p>
    {:catch main_error}
        <Alert
            color="danger"
            fade={false}
        >
            <strong>{$t('errors.domain-import')}</strong>
            {main_error}
        </Alert>
    {/await}
</div>
