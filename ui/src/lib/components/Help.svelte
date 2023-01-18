<script lang="ts">
 import {
     Button,
     Icon,
 } from 'sveltestrap';
 import type {
     ButtonColor,
     ButtonSize,
 } from 'sveltestrap/src/Button.d';

 import { t } from '$lib/translations';

 export { className as class };
 let className: string = '';
 export let color: ButtonColor = "primary";
 export let href: string;
 export let size: ButtonSize | undefined = undefined;
</script>

<Button
    {href}
    target="_blank"
    {color}
    {size}
    class="umami--click--help {className}"
>
    <Icon
        name="question-circle-fill"
        title={$t('common.help')}
    />
</Button>
