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
        Form,
        FormGroup,
        Input,
        Label,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { setAdminToken } from "$lib/stores/adminsession";
    import { navigate } from "$lib/stores/config";

    let password = $state("");
    let duration = $state(3600);
    let loading = $state(false);
    let error = $state("");

    // nextTarget returns the same-origin path to redirect to after login,
    // falling back to the dashboard. The candidate is resolved against the
    // current origin and rejected unless it stays on it: a prefix check alone
    // is not enough, because browsers fold backslashes into slashes, so
    // "/\evil.com" passes a `!startsWith("//")` test yet resolves off-site.
    function nextTarget(): string {
        const next = new URLSearchParams(window.location.search).get("next");
        if (!next) return "/";

        try {
            const resolved = new URL(next, window.location.origin);
            if (resolved.origin === window.location.origin) {
                return resolved.pathname + resolved.search + resolved.hash;
            }
        } catch {
            // Malformed target: fall through to the dashboard.
        }

        return "/";
    }

    async function submit(event: Event) {
        event.preventDefault();
        error = "";
        loading = true;

        try {
            const response = await fetch("/api/admin-login", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ password, duration }),
            });

            if (!response.ok) {
                const body = await response.json().catch(() => ({}));
                error = body.errmsg || "Login failed.";
                return;
            }

            const body = await response.json();
            setAdminToken(body.token, body.expires_at);
            navigate(nextTarget());
        } catch (err) {
            error = (err as Error).message || "Login failed.";
        } finally {
            loading = false;
        }
    }
</script>

<div class="container" style="max-width: 25rem;">
    <Card>
        <CardBody>
            <h1 class="h4 text-center mb-3">Admin sign in</h1>

            {#if error}
                <Alert color="danger">{error}</Alert>
            {/if}

            <Form on:submit={submit}>
                <FormGroup>
                    <Label for="admin-password">Password</Label>
                    <Input
                        id="admin-password"
                        type="password"
                        autocomplete="current-password"
                        bind:value={password}
                        disabled={loading}
                        required
                    />
                </FormGroup>

                <FormGroup>
                    <Label for="admin-duration">Session duration</Label>
                    <Input id="admin-duration" type="select" bind:value={duration} disabled={loading}>
                        <option value={3600}>1 hour</option>
                        <option value={28800}>8 hours</option>
                        <option value={86400}>24 hours</option>
                    </Input>
                </FormGroup>

                <Button type="submit" color="primary" class="w-100" disabled={loading || !password}>
                    {#if loading}
                        <Spinner size="sm" />
                    {/if}
                    Sign in
                </Button>
            </Form>
        </CardBody>
    </Card>
</div>
