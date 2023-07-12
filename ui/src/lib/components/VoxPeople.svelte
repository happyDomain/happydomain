<script lang="ts">
 import {
     Icon,
 } from 'sveltestrap';

 const { MODE } = import.meta.env;

 import { userSession } from '$lib/stores/usersession';
 import { t, locale } from '$lib/translations';

 export let routeId: string | null;
 const instancename = encodeURIComponent(window.location.hostname);
</script>

{#if MODE == 'production'}
    <a
        id="voxpeople"
        href="https://framaforms.org/quel-est-votre-avis-sur-happydns-1610366701?u={$userSession?$userSession.id:0}&amp;i={instancename}&amp;p={routeId}&amp;l={$locale}"
        target="_blank"
        rel="noreferrer"
        title={$t('common.survey')}
        class="d-flex btn btn-light justify-content-center align-items-center"
        data-umami-event="vox-people"
    >
        <Icon
            name="chat-right-text"
        />
    </a>
{/if}

<style>
 #voxpeople {
     position: fixed;
     bottom: 7vh;
     right: 4vw;
     z-index: 1051;
     height: max(4vw, 4vh);
     width: max(4vw, 4vh);
     border-radius: 4vw;
     box-shadow: 0 0px 3px 0 #9332bb;
     font-size: max(1.7vw, 1.7vh);
 }
</style>
