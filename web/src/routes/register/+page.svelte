<script lang="ts">
    import { resolve } from "$app/paths";

    import AuthCard from "$lib/components/AuthCard.svelte";
    import { t } from "$lib/translations";
    import SignUpForm from "./SignUpForm.svelte";
</script>

<svelte:head>
    <title>{$t("account.join")} - happyDomain</title>
</svelte:head>

<AuthCard
    src="/img/signup.webp"
    alt="A person with a backpack walking into a greenhouse, overlooking a green city"
>
    <h1 class="h2 mb-2">{$t("account.signup.title")}</h1>
    <p class="text-body-secondary mb-4">{$t("account.signup.join-call")}</p>

    <SignUpForm />

    <div class="mt-4 pt-3 border-top">
        <span class="text-body-secondary">{$t("account.signup.already")}</span>
        <a href={resolve("/login")} class="fw-semibold">
            {$t("menu.signin")}
        </a>
    </div>
</AuthCard>
