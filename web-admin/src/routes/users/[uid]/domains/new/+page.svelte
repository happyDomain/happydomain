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
        Col,
        Container,
        Form,
        FormGroup,
        Icon,
        Input,
        Label,
        Row,
        Spinner,
    } from "@sveltestrap/sveltestrap";

    import { postDomains } from '$lib/api-admin';
    import { toasts } from '$lib/stores/toasts';

    let domain = $state('');
    let group = $state('');
    let id_owner = $state('');
    let id_provider = $state('');
    let loading = $state(false);
    let errorMessage = $state('');

    async function handleSubmit() {
        loading = true;
        errorMessage = '';

        try {
            const response = await postDomains({
                body: {
                    domain: domain,
                    group: group || undefined,
                    id_owner: id_owner || undefined,
                    id_provider: id_provider || undefined,
                }
            });

            toasts.addToast({
                message: `Domain "${domain}" has been created successfully`,
                type: 'success',
                timeout: 5000,
            });

            goto('/domains');
        } catch (error) {
            errorMessage = 'Failed to create domain: ' + error;
            toasts.addErrorToast({
                message: errorMessage,
                timeout: 10000,
            });
        } finally {
            loading = false;
        }
    }
</script>

<Container class="flex-fill my-5">
    <Row class="mb-4">
        <Col>
            <h1 class="display-5">
                <Icon name="plus-circle"></Icon>
                Create New Domain
            </h1>
        </Col>
    </Row>

    <Row>
        <Col md={8} lg={6}>
            <Card>
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
                            <Label for="domain">Domain Name (FQDN) *</Label>
                            <Input
                                type="text"
                                id="domain"
                                bind:value={domain}
                                required
                                placeholder="example.com"
                                autofocus
                            />
                            <small class="form-text text-muted">
                                The fully qualified domain name to manage.
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
                            <Input
                                type="text"
                                id="id_owner"
                                bind:value={id_owner}
                                placeholder="owner-id"
                            />
                            <small class="form-text text-muted">
                                The identifier of the domain's owner.
                            </small>
                        </FormGroup>

                        <FormGroup>
                            <Label for="id_provider">Provider ID</Label>
                            <Input
                                type="text"
                                id="id_provider"
                                bind:value={id_provider}
                                placeholder="provider-id"
                            />
                            <small class="form-text text-muted">
                                The identifier of the provider used to access and edit the domain.
                            </small>
                        </FormGroup>

                        <div class="d-flex gap-2 mt-4">
                            <Button color="primary" type="submit" disabled={loading}>
                                {#if loading}
                                    <Spinner size="sm" class="me-2" />
                                {:else}
                                    <Icon name="plus-circle" class="me-2"></Icon>
                                {/if}
                                Create Domain
                            </Button>
                            <Button type="button" color="secondary" outline href="/domains" disabled={loading}>
                                Cancel
                            </Button>
                        </div>
                    </Form>
                </CardBody>
            </Card>
        </Col>
    </Row>
</Container>
