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

    import { postAuth } from '$lib/api-admin';
    import { toasts } from '$lib/stores/toasts';

    let email = $state('');
    let loading = $state(false);
    let errorMessage = $state('');

    async function handleSubmit() {
        loading = true;
        errorMessage = '';

        try {
            const response = await postAuth({
                body: {
                    email: email,
                }
            });

            toasts.addToast({
                message: `Auth user "${email}" has been created successfully`,
                type: 'success',
                timeout: 5000,
            });

            goto('/auth_users');
        } catch (error) {
            errorMessage = 'Failed to create auth user: ' + error;
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
                Create New Auth User
            </h1>
        </Col>
    </Row>

    <Row>
        <Col md={8} lg={6}>
            <Card>
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
                            <Label for="email">Email *</Label>
                            <Input
                                type="email"
                                id="email"
                                bind:value={email}
                                required
                                placeholder="user@example.com"
                                autofocus
                            />
                            <small class="form-text text-muted">
                                The email address will be used as the authentication account's login.
                            </small>
                        </FormGroup>

                        <div class="d-flex gap-2 mt-4">
                            <Button color="primary" type="submit" disabled={loading}>
                                {#if loading}
                                    <Spinner size="sm" class="me-2" />
                                {:else}
                                    <Icon name="plus-circle" class="me-2"></Icon>
                                {/if}
                                Create Auth User
                            </Button>
                            <Button type="button" color="secondary" outline href="/auth_users" disabled={loading}>
                                Cancel
                            </Button>
                        </div>
                    </Form>
                </CardBody>
            </Card>
        </Col>
    </Row>
</Container>
