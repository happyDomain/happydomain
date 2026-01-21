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
        InputGroup,
        Label,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { putUsersByUidDomainsByDomain } from '$lib/api-admin';
    import { toasts } from '$lib/stores/toasts';

    interface Props {
        domainData: any;
        uid: string;
        domainId: string;
    }

    let { domainData, uid, domainId }: Props = $props();

    let domainName = $state('');
    let group = $state('');
    let id_owner = $state('');
    let id_provider = $state('');
    let loading = $state(false);
    let errorMessage = $state('');

    // Update local state when domainData changes
    $effect(() => {
        if (domainData) {
            domainName = domainData.domain || '';
            group = domainData.group || '';
            id_owner = domainData.id_owner || '';
            id_provider = domainData.id_provider || '';
        }
    });

    async function handleSubmit(e: Event) {
        e.preventDefault();

        loading = true;
        errorMessage = '';

        try {
            await putUsersByUidDomainsByDomain({
                path: { uid, domain: domainId },
                body: {
                    domain: domainName,
                    group: group || undefined,
                    id_owner: id_owner || undefined,
                    id_provider: id_provider || undefined,
                }
            });

            toasts.addToast({
                message: `Domain "${domainName}" has been updated successfully`,
                type: 'success',
                timeout: 5000,
            });

            goto('/domains');
        } catch (error) {
            errorMessage = 'Failed to update domain: ' + error;
            toasts.addErrorToast({
                message: errorMessage,
                timeout: 10000,
            });
        } finally {
            loading = false;
        }
    }

    function handleCancel() {
        goto('/domains');
    }
</script>

<Card class="mb-4">
    <CardHeader>
        <h5 class="mb-0">Domain Information</h5>
    </CardHeader>
    <CardBody>
        {#if errorMessage}
            <Alert color="danger" dismissible fade>
                {errorMessage}
            </Alert>
        {/if}

        <Form on:submit={handleSubmit}>
            <FormGroup>
                <Label for="domainId">Domain ID</Label>
                <Input
                    type="text"
                    id="domainId"
                    value={domainData?.id || ''}
                    disabled
                    readonly
                />
            </FormGroup>

            <FormGroup>
                <Label for="domainName">Domain Name (FQDN) *</Label>
                <Input
                    type="text"
                    id="domainName"
                    bind:value={domainName}
                    required
                    placeholder="example.com"
                />
                <small class="form-text text-muted">
                    The fully qualified domain name.
                </small>
            </FormGroup>

            <FormGroup>
                <Label for="group">Group</Label>
                <Input
                    type="text"
                    id="group"
                    bind:value={group}
                    placeholder="production"
                />
                <small class="form-text text-muted">
                    Optional hint string to group domains together.
                </small>
            </FormGroup>

            <FormGroup>
                <Label for="id_owner">Owner ID</Label>
                <InputGroup>
                    <Input
                        type="text"
                        id="id_owner"
                        bind:value={id_owner}
                        placeholder="owner-id"
                    />
                    <Button
                        color="secondary"
                        outline
                        href="/users/{id_owner}"
                        disabled={!id_owner}
                    >
                        <Icon name="arrow-right"></Icon>
                    </Button>
                </InputGroup>
            </FormGroup>

            <FormGroup>
                <Label for="id_provider">Provider ID</Label>
                <InputGroup>
                    <Input
                        type="text"
                        id="id_provider"
                        bind:value={id_provider}
                        placeholder="provider-id"
                    />
                    <Button
                        color="secondary"
                        outline
                        href="/providers/{id_provider}"
                        disabled={!id_provider}
                    >
                        <Icon name="arrow-right"></Icon>
                    </Button>
                </InputGroup>
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
                <Button
                    type="button"
                    color="secondary"
                    outline
                    onclick={handleCancel}
                    disabled={loading}
                >
                    Cancel
                </Button>
            </div>
        </Form>
    </CardBody>
</Card>
