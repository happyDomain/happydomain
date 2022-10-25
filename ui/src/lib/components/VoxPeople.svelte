<script>
 import {
     Icon,
 } from 'sveltestrap';

 import { t, locale } from '$lib/translations';

 const user_isLogged = false;
 const instancename = "";
 const router = {history:{current:{}}};
</script>

<a
    id="voxpeople"
    href="https://framaforms.org/quel-est-votre-avis-sur-happydns-1610366701?u={user_isLogged?user_getSession.id:0}&amp;i={instancename}&amp;p={router.history.current.name}&amp;l={$locale}"
    target="_blank"
    title={$t('common.survey')}
    class="d-flex justify-content-center align-items-center umami--click--vox-people"
>
    <Icon
        name="chat-right-text"
    />
</a>
