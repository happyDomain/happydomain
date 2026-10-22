<script lang="ts">
    import { resolve } from "$app/paths";

    import AuthCard from "$lib/components/AuthCard.svelte";
    import { appConfig } from "$lib/stores/config";
    import { t } from "$lib/translations";
    import LoginForm from "./LoginForm.svelte";
</script>

<svelte:head>
    <title>{$t("onboarding.steps.welcome")} - HappyDomain</title>
</svelte:head>

<AuthCard
    src="/img/login.webp"
    alt="A desk with a laptop and a mug in a greenhouse, overlooking a green city"
    mobilePosition="center 65%"
>
    <h1 class="h2 mb-2">{$t("account.welcome")}</h1>
    <p class="text-body-secondary mb-4">{$t("account.login-call")}</p>

    <LoginForm />

    {#if !$appConfig.disable_registration}
        <div class="mt-4 pt-3 border-top">
            <span class="text-body-secondary">{$t("account.ask-have")}</span>
            <a href={resolve("/register")} class="fw-semibold">
                {$t("account.join")}
            </a>
        </div>
    {/if}
</AuthCard>
