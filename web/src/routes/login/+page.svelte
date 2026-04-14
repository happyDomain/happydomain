<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2024 happyDomain
     Authors: Pierre-Olivier Mercier, et al.

     This program is offered under a commercial and under the AGPL license.
     For commercial licensing, contact us at <contact@happydomain.org>.

     For AGPL licensing:
     This program is free software: you can redistribute it and/or modify
     it under the terms of the GNU Affero General Public License as published by
     the Free Software Foundation, either version 3 of the License, or
     (at your option) any later version.

     This program is distributed in the hope that it will be useful,
     but WITHOUT ANY WARRANTY; without even the implied warranty of
     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
     GNU Affero General Public License for more details.

     You should have received a copy of the GNU Affero General Public License
     along with this program.  If not, see <https://www.gnu.org/licenses/>.
-->

<script lang="ts">
    import { Col, Container, Row } from "@sveltestrap/sveltestrap";

    import Logo from "$lib/components/Logo.svelte";
    import { appConfig } from "$lib/stores/config";
    import { t } from "$lib/translations";
    import LoginForm from "./LoginForm.svelte";
</script>

<svelte:head>
    <title>{$t("onboarding.steps.welcome")} - happyDomain</title>
</svelte:head>

<Container class="my-auto py-4">
    <Row class="justify-content-center align-items-stretch g-0">
        <Col md="5" lg="4" class="d-none d-md-block">
            <div class="login-visual" title="Welcome back!"></div>
        </Col>
        <Col sm="10" md="7" lg="5">
            <div class="login-card">
                <div class="text-center mb-4 d-md-none">
                    <Logo height="32" />
                </div>
                <h5 class="fw-bold mb-4">{$t("account.signup.join-call")}</h5>
                <LoginForm />
                {#if !$appConfig.disable_registration}
                    <div class="text-center mt-4 pt-3 border-top">
                        <span class="text-body-secondary">{$t("account.ask-have")}</span>
                        <a href="/register" class="fw-semibold text-decoration-none ms-1">
                            {$t("account.join")}
                        </a>
                    </div>
                {/if}
            </div>
        </Col>
    </Row>
</Container>

<style>
    .login-visual {
        height: 100%;
        border-radius: 1rem 0 0 1rem;
        overflow: hidden;
        background: url(/img/login.webp), linear-gradient(135deg, #f0faf7 0%, #e8f4f8 100%);
        background-size: cover;
        background-position: center;
        min-height: 560px;
        display: flex;
        align-items: center;
        justify-content: center;
    }

    .login-card {
        background: #fff;
        border-radius: 0 1rem 1rem 0;
        padding: 2.5rem 2rem;
        box-shadow: 0 4px 24px rgba(0, 0, 0, 0.08);
        height: 100%;
        display: flex;
        flex-direction: column;
        justify-content: center;
    }

    @media (max-width: 767.98px) {
        .login-card {
            border-radius: 1rem;
        }
    }
</style>
