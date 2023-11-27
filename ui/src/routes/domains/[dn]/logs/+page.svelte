<script lang="ts">
 import {
     Button,
     Icon,
     Table,
     Spinner,
 } from 'sveltestrap';

 import { getDomainLogs } from '$lib/api/domains';
 import { getUser } from '$lib/stores/users';
 import { t } from '$lib/translations';

 export let data: {domain: DomainInList; history: string; streamed: Object;};
</script>

<div class="flex-fill pb-4 pt-2">
    <h2>Journal du domaine <span class="font-monospace">{data.domain.domain}</span></h2>
    {#await getDomainLogs(data.domain.id)}
        <div class="mt-5 text-center flex-fill">
            <Spinner label="Spinning" />
            <p>{$t('wait.loading')}</p>
        </div>
    {:then logs}
        <Table hover stripped>
            <thead>
                <tr>
                    <th>Utilisateur</th>
                    <th>Action/description</th>
                    <th>Date</th>
                    <th>Niveau</th>
                </tr>
            </thead>
            <tbody>
                {#if logs}
                    {#each logs as log}
                        <tr>
                            <td>
                                {#await getUser(log.id_user)}
                                    {log.id_user}
                                {:then user}
                                    <span title={user.Email}>
                                        {user.Email}
                                    </span>
                                {/await}
                            </td>
                            <td>{log.content}</td>
                            <td>
                                {new Intl.DateTimeFormat(undefined, {dateStyle: "short", timeStyle: "medium"}).format(new Date(log.date))}
                            </td>
                            <td>{log.level}</td>
                        </tr>
                    {/each}
                {:else}
                    <tr>
                        <td colspan="4" class="text-center">
                            Aucune entrée dans le journal du domaine.
                        </td>
                    </tr>
                {/if}
            </tbody>
        </Table>
    {/await}
</div>