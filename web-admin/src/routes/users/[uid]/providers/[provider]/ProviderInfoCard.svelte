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
        Badge,
        Button,
        Card,
        CardBody,
        CardHeader,
        Form,
        FormGroup,
        Icon,
        Input,
        InputGroup,
        Label,
        Modal,
        ModalBody,
        ModalFooter,
        ModalHeader,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { putUsersByUidProvidersByPid, deleteUsersByUidProvidersByPid } from '$lib/api-admin';
    import type { HappydnsProvider } from '$lib/api-admin';
    import { toasts } from '$lib/stores/toasts';

    interface ProviderInfoCardProps {
        provider: HappydnsProvider;
        uid: string;
    }

    let { provider, uid }: ProviderInfoCardProps = $props();

    let comment = $state('');
    let loading = $state(false);
    let errorMessage = $state('');
    let showDeleteModal = $state(false);
    let deleting = $state(false);

    // Load provider data when provider changes
    $effect(() => {
        if (provider) {
            comment = provider._comment || '';
        }
    });

    async function handleSubmit(e: SubmitEvent) {
        e.preventDefault();

        loading = true;
        errorMessage = '';

        try {
            const body: HappydnsProvider = {
                ...provider,
                _comment: comment,
            };

            await putUsersByUidProvidersByPid({
                path: { uid, pid: provider._id! },
                body: body
            });

            toasts.addToast({
                message: `Provider "${comment || provider._srctype}" has been updated successfully`,
                type: 'success',
                timeout: 5000,
            });

            goto(`/users/${uid}`);
        } catch (error) {
            errorMessage = 'Failed to update provider: ' + error;
            toasts.addErrorToast({
                message: errorMessage,
                timeout: 10000,
            });
        } finally {
            loading = false;
        }
    }

    async function handleDelete() {
        deleting = true;
        errorMessage = '';

        try {
            await deleteUsersByUidProvidersByPid({
                path: { uid, pid: provider._id! }
            });

            toasts.addToast({
                message: `Provider "${comment || provider._srctype}" has been deleted successfully`,
                type: 'success',
                timeout: 5000,
            });

            showDeleteModal = false;
            goto(`/users/${uid}`);
        } catch (error) {
            errorMessage = 'Failed to delete provider: ' + error;
            toasts.addErrorToast({
                message: errorMessage,
                timeout: 10000,
            });
            showDeleteModal = false;
        } finally {
            deleting = false;
        }
    }
</script>

<Card class="mb-4">
    <CardHeader>
        <div class="d-flex justify-content-between align-items-center">
            <h5 class="mb-0">Provider Information</h5>
            {#if provider._srctype}
                <Badge color="info">{provider._srctype}</Badge>
            {/if}
        </div>
    </CardHeader>
    <CardBody>
        {#if errorMessage}
            <Alert color="danger" dismissible fade>
                {errorMessage}
            </Alert>
        {/if}

        <Form on:submit={handleSubmit}>
            <FormGroup>
                <Label for="providerId">Provider ID</Label>
                <Input
                    type="text"
                    id="providerId"
                    value={provider._id}
                    disabled
                    readonly
                />
            </FormGroup>

            <FormGroup>
                <Label for="providerType">Provider Type</Label>
                <Input
                    type="text"
                    id="providerType"
                    value={provider._srctype}
                    disabled
                    readonly
                />
            </FormGroup>

            <FormGroup>
                <Label for="ownerId">Owner ID</Label>
                <InputGroup>
                    <Input
                        type="text"
                        id="ownerId"
                        value={provider._ownerid}
                        disabled
                        readonly
                    />
                    <Button
                        color="secondary"
                        outline
                        href="/users/{provider._ownerid}"
                    >
                        <Icon name="person"></Icon>
                    </Button>
                </InputGroup>
            </FormGroup>

            <FormGroup>
                <Label for="comment">Comment / Description</Label>
                <Input
                    type="text"
                    id="comment"
                    bind:value={comment}
                    placeholder="e.g., OVH Production"
                />
                <small class="text-muted">
                    A description to help identify this provider
                </small>
            </FormGroup>

            <div class="d-flex gap-2 mt-4">
                <Button color="primary" type="submit" disabled={loading || deleting}>
                    {#if loading}
                        <Spinner size="sm" class="me-2" />
                    {:else}
                        <Icon name="check-circle" class="me-2"></Icon>
                    {/if}
                    Save Changes
                </Button>
                <Button type="button" color="secondary" outline href="/users/{uid}" disabled={loading || deleting}>
                    <Icon name="arrow-left" class="me-2"></Icon>
                    Back to User
                </Button>
                <div class="ms-auto">
                    <Button
                        type="button"
                        color="danger"
                        outline
                        onclick={() => showDeleteModal = true}
                        disabled={loading || deleting}
                    >
                        <Icon name="trash" class="me-2"></Icon>
                        Delete Provider
                    </Button>
                </div>
            </div>
        </Form>
    </CardBody>
</Card>

<Modal isOpen={showDeleteModal} toggle={() => showDeleteModal = false}>
    <ModalHeader toggle={() => showDeleteModal = false}>
        Confirm Deletion
    </ModalHeader>
    <ModalBody>
        <p>Are you sure you want to delete this provider?</p>
        <Alert color="warning" class="mb-0">
            <strong>Warning:</strong> This action cannot be undone. Deleting this provider may affect domains using it.
        </Alert>
    </ModalBody>
    <ModalFooter>
        <Button color="secondary" onclick={() => showDeleteModal = false} disabled={deleting}>
            Cancel
        </Button>
        <Button color="danger" onclick={handleDelete} disabled={deleting}>
            {#if deleting}
                <Spinner size="sm" class="me-2" />
            {:else}
                <Icon name="trash" class="me-2"></Icon>
            {/if}
            Delete Provider
        </Button>
    </ModalFooter>
</Modal>
