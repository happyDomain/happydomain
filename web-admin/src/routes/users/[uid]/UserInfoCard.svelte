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

    import { putUsersByUid } from '$lib/api-admin';
    import { toasts } from '$lib/stores/toasts';
    import { toDatetimeLocal, fromDatetimeLocal } from '$lib/utils';

    import type { HappydnsUser } from '$lib/api-admin';

    interface UserInfoCardProps {
        user: HappydnsUser;
        uid: string;
    }

    let { user, uid }: UserInfoCardProps = $props();

    let email = $state('');
    let createdAt = $state('');
    let lastSeen = $state('');
    let loading = $state(false);
    let errorMessage = $state('');

    // Load user data when user changes
    $effect(() => {
        if (user) {
            email = user.email || '';
            createdAt = toDatetimeLocal(user.created_at);
            lastSeen = toDatetimeLocal(user.last_seen);
        }
    });

    async function handleSubmit(e: Event) {
        e.preventDefault();

        loading = true;
        errorMessage = '';

        try {
            const body: any = {
                email: email,
                created_at: fromDatetimeLocal(createdAt),
            };

            // Only include last_seen if it has a value
            if (lastSeen) {
                body.last_seen = fromDatetimeLocal(lastSeen);
            } else {
                body.last_seen = null;
            }

            await putUsersByUid({
                path: { uid },
                body: body
            });

            toasts.addToast({
                message: `User "${email}" has been updated successfully`,
                type: 'success',
                timeout: 5000,
            });

            goto('/users');
        } catch (error) {
            errorMessage = 'Failed to update user: ' + error;
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
        <h5 class="mb-0">User Information</h5>
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
                    value={user.id}
                    disabled
                    readonly
                />
            </FormGroup>

            <FormGroup>
                <Label for="email">Email *</Label>
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
                <Label for="lastSeen">Last Seen</Label>
                <Input
                    type="datetime-local"
                    id="lastSeen"
                    bind:value={lastSeen}
                />
            </FormGroup>

            <div class="d-flex gap-2 mt-4">
                <Button color="primary" type="submit" disabled={loading}>
                    {#if loading}
                        <Spinner size="sm" class="me-2" />
                    {:else}
                        <Icon name="check-circle" class="me-2"></Icon>
                    {/if}
                    Save Changes
                </Button>
                <Button type="button" color="secondary" outline href="/users" disabled={loading}>
                    Cancel
                </Button>
            </div>
        </Form>
    </CardBody>
</Card>
