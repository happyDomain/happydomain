<script lang="ts">
 import { goto } from '$app/navigation';

 import {
     Badge,
     Button,
     Card,
     CardBody,
     CardGroup,
     Col,
     Container,
     Icon,
     Row,
     Spinner,
 } from 'sveltestrap';

 import Logo from '$lib/components/Logo.svelte';
 import ProviderList from '$lib/components/providers/List.svelte';
 import ProviderSelector from '$lib/components/providers/Selector.svelte';
 import { providers, refreshProviders } from '$lib/stores/providers';
 import { t } from '$lib/translations';

 if (!$providers) refreshProviders();
</script>

<Container class="pt-3 pb-4">
    <h1 class="text-center mb-4">
        {$t('common.welcome.start')}<Logo height="40" />{$t('common.welcome.end')}
    </h1>
    <CardGroup class="my-4">
        <Card>
            <CardBody>
                <h3 class="text-secondary text-center mt-1 mb-4">
                    {$t('onboarding.no-sale.title')}
                </h3>
                <p class="text-justify text-indent mt-4 mb-3">
                    {@html $t('onboarding.no-sale.description', {"happyDomain": `happy<span class="fw-bold">Domain</span>`})}
                </p>
                <p class="text-justify text-indent mt-3 mb-4">
                    {$t('onboarding.no-sale.buy-advice')}
                </p>
            </CardBody>
        </Card>
        <Card>
            <CardBody>
                <h3 class="text-primary text-center mt-1 mb-4">
                    {$t('onboarding.own')}
                </h3>
                <p class="text-justify text-indent my-4">
                    {@html $t('onboarding.use', {
                        "happyDomain": `happy<span class="fw-bold">Domain</span>`,
                        "first-step": $providers && $providers.length ? $t('onboarding.choose-configured', {"action": `<a href="/providers/new">${$t('onboarding.add-one')}</a>`}) : $t('onboarding.suggest-provider')
                      })}
                </p>
                {#if $providers && $providers.length}
                    <ProviderList
                        items={$providers}
                        noLabel
                        style="max-height: 20rem; overflow-y: auto"
                        on:click={(event) => goto(`/providers/${event.detail._id}/domains`)}
                        on:new-provider={() => goto(`/providers/new`)}
                    />
                {:else}
                    <div style="max-height: 14rem; overflow-y: auto;">
                        <ProviderSelector
                            on:provider-selected={(event) => goto(`/providers/new/${event.detail.ptype}`)}
                        />
                    </div>
                {/if}
            </CardBody>
        </Card>
    </CardGroup>

    <Card id="aa-hosting" class="my-3">
        <CardBody>
            <span class="text-secondary fw-bold">{$t('onboarding.questions.hosting.q')}</span><br>
            <div class="mx-3">
                {$t('onboarding.questions.hosting.a')}
            </div>
        </CardBody>
    </Card>

    <Card id="sec-hosting" class="my-3">
        <CardBody>
            <span class="text-secondary fw-bold">
                {@html $t('onboarding.questions.secondary.q', {"happyDomain": `happy<span class="fw-bold">Domain</span>`})}
            </span>
            <br>
            <div class="mx-3">
                {$t('onboarding.questions.secondary.a')}
            </div>
        </CardBody>
    </Card>
</Container>
