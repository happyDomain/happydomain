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
    import { goto } from '$app/navigation';
    import {
        Alert,
        Button,
        Card,
        CardBody,
        CardHeader,
        Form,
        FormGroup,
        Icon,
        Input,
        Label,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import type { HappydnsUserAuth } from '$lib/api-admin';
    import { putAuthByUid } from '$lib/api-admin';
    import { toasts } from '$lib/stores/toasts';
    import { toDatetimeLocal, fromDatetimeLocal } from '$lib/utils';

    interface Props {
        authUser: HappydnsUserAuth;
    }

    let {
        authUser,
    }: Props = $props();

    let email = $state('');
    let createdAt = $state('');
    let lastLoggedIn = $state('');
    let emailVerification = $state('');
    let allowCommercials = $state(false);
    let loading = $state(false);
    let errorMessage = $state('');

    // Load auth user data
    $effect(() => {
        email = authUser.email || '';
        createdAt = toDatetimeLocal(authUser.createdAt);
        lastLoggedIn = toDatetimeLocal(authUser.lastLoggedIn);
        emailVerification = toDatetimeLocal(authUser.emailVerification);
        allowCommercials = authUser.allowCommercials || false;
    });

    async function handleSubmit(e: SubmitEvent) {
        e.preventDefault();

        loading = true;
        errorMessage = '';

        try {
            const body: any = {
                email: email,
                createdAt: fromDatetimeLocal(createdAt),
                allowCommercials: allowCommercials,
            };

            // Only include optional fields if they have values
            if (lastLoggedIn) {
                body.lastLoggedIn = fromDatetimeLocal(lastLoggedIn);
            } else {
                body.lastLoggedIn = null;
            }

            if (emailVerification) {
                body.emailVerification = fromDatetimeLocal(emailVerification);
            } else {
                body.emailVerification = null;
            }

            await putAuthByUid({
                path: { uid: authUser.id ?? '' },
                body: body
            });

            toasts.addToast({
                message: `Auth user "${email}" has been updated successfully`,
                type: 'success',
                timeout: 5000,
            });

            goto('/auth_users');
        } catch (error) {
            errorMessage = 'Failed to update auth user: ' + error;
            toasts.addErrorToast({
                message: errorMessage,
                timeout: 10000,
            });
        } finally {
            loading = false;
        }
    }
</script>

<Card class="mb-4">
    <CardHeader>
        <h5 class="mb-0">Auth User Information</h5>
    </CardHeader>
    <CardBody>
        {#if errorMessage}
            <Alert color="danger" dismissible fade>
                {errorMessage}
            </Alert>
        {/if}

        <Form on:submit={handleSubmit}>
            <FormGroup>
                <Label for="userId">User ID</Label>
                <Input
                    type="text"
                    id="userId"
                    value={authUser.id}
                    disabled
                    readonly
                />
            </FormGroup>

            <FormGroup>
                <Label for="email">Email</Label>
                <Input
                    type="email"
                    id="email"
                    bind:value={email}
                    required
                    placeholder="user@example.com"
                />
            </FormGroup>

            <FormGroup>
                <Label for="createdAt">Created At</Label>
                <Input
                    type="datetime-local"
                    id="createdAt"
                    bind:value={createdAt}
                />
            </FormGroup>

            <FormGroup>
                <Label for="lastLoggedIn">Last Logged In</Label>
                <Input
                    type="datetime-local"
                    id="lastLoggedIn"
                    bind:value={lastLoggedIn}
                />
            </FormGroup>

            <FormGroup>
                <Label for="emailVerification">Email Verification</Label>
                <Input
                    type="datetime-local"
                    id="emailVerification"
                    bind:value={emailVerification}
                    placeholder="Not verified"
                />
            </FormGroup>

            <Input
                type="checkbox"
                id="allowCommercials"
                bind:checked={allowCommercials}
                label="Allow Commercial Communications"
            />

            <div class="d-flex gap-2 mt-4">
                <Button color="primary" type="submit" disabled={loading}>
                    {#if loading}
                        <Spinner size="sm" class="me-2" />
                    {:else}
                        <Icon name="check-circle" class="me-2"></Icon>
                    {/if}
                    Save Changes
                </Button>
                <Button type="button" color="secondary" outline href="/auth_users" disabled={loading}>
                    Cancel
                </Button>
            </div>
        </Form>
    </CardBody>
</Card>
