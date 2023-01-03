<script lang="ts">
 import { page } from '$app/stores'

 import {
     Button,
     Icon,
 } from 'sveltestrap';

 import { t, locale } from '$lib/translations';

 export { className as class };
 let className = '';
 export let color = "primary"
 export let href: string;
 export let size: string | null;
</script>

<Button
    {href}
    target="_blank"
    {color}
    {size}
    class={className}
>
    <Icon
        name="question-circle-fill"
        title={$t('common.help')}
    />
</Button>
