<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2026 happyDomain
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
    import {
        Alert,
        Button,
        Card,
        CardBody,
        CardHeader,
        Icon,
        Input,
        InputGroup,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { getAuthByUid } from "$lib/api-admin";
    import type { HappydnsUserAuth } from "$lib/api-admin";
    import { toasts } from "$lib/stores/toasts";

    interface UserCreateAuthCardProps {
        uid: string;
        email: string;
    }

    let { uid, email }: UserCreateAuthCardProps = $props();

    let loading = $state(false);
    let checking = $state(true);
    let existingAuth = $state<HappydnsUserAuth | null>(null);
    let generatedPassword = $state("");
    let errorMessage = $state("");

    $effect(() => {
        if (!email) {
            checking = false;
            return;
        }
        checking = true;
        existingAuth = null;
        getAuthByUid({ path: { uid: email } })
            .then((res) => {
                existingAuth = (res?.data as HappydnsUserAuth) ?? null;
            })
            .catch(() => {
                existingAuth = null;
            })
            .finally(() => {
                checking = false;
            });
    });

    async function createAuth() {
        if (
            !confirm(
                `Create an authentication account for "${email}"? A new password will be generated.`,
            )
        ) {
            return;
        }

        loading = true;
        errorMessage = "";
        generatedPassword = "";

        try {
            const res = await fetch(`/api/users/${encodeURIComponent(uid)}/new_auth`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
            });

            const data = await res.json();
            if (!res.ok) {
                throw new Error(data?.error || data?.message || res.statusText);
            }

            generatedPassword = data.password || "";
            existingAuth = (data.authUser as HappydnsUserAuth) ?? existingAuth;
            toasts.addToast({
                message: `Auth account created for "${email}".`,
                type: "success",
                timeout: 5000,
            });
        } catch (err) {
            errorMessage = "Failed to create auth account: " + err;
            toasts.addErrorToast({ message: errorMessage, timeout: 10000 });
        } finally {
            loading = false;
        }
    }

    async function copyPassword() {
        try {
            await navigator.clipboard.writeText(generatedPassword);
            toasts.addToast({
                message: "Password copied to clipboard.",
                type: "success",
                timeout: 3000,
            });
        } catch (err) {
            toasts.addErrorToast({
                message: "Unable to copy password: " + err,
                timeout: 5000,
            });
        }
    }
</script>

<Card class="mb-4">
    <CardHeader>
        <h5 class="mb-0">Authentication account</h5>
    </CardHeader>
    <CardBody>
        {#if errorMessage}
            <Alert color="danger" dismissible fade>{errorMessage}</Alert>
        {/if}

        {#if generatedPassword}
            <Alert color="success">
                <p class="mb-2">
                    <Icon name="check-circle" class="me-1" />
                    Auth account created. Share this password with the user — it
                    won't be shown again.
                </p>
                <InputGroup>
                    <Input type="text" value={generatedPassword} readonly />
                    <Button color="secondary" outline on:click={copyPassword}>
                        <Icon name="clipboard" />
                    </Button>
                </InputGroup>
            </Alert>
        {:else if checking}
            <div class="text-center text-muted">
                <Spinner size="sm" class="me-2" />
                Checking existing auth account…
            </div>
        {:else if existingAuth}
            <p class="mb-2">
                An authentication account already exists for
                <strong>{existingAuth.email}</strong>.
            </p>
            <Button
                color="secondary"
                outline
                href={`/auth_users/${existingAuth.id}`}
            >
                <Icon name="box-arrow-up-right" class="me-2" />
                Open auth account
            </Button>
        {:else}
            <p class="text-muted small mb-3">
                Create a <code>UserAuth</code> tied to this user's email so they
                can log in with a password. A random password will be generated
                and shown once.
            </p>
            <Button color="primary" on:click={createAuth} disabled={loading}>
                {#if loading}
                    <Spinner size="sm" class="me-2" />
                {:else}
                    <Icon name="key" class="me-2" />
                {/if}
                Create auth account
            </Button>
        {/if}
    </CardBody>
</Card>
