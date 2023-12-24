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
 import { goto } from '$app/navigation';

 import {
     Card,
     CardBody,
     CardHeader,
     Container,
     Col,
     Row,
 } from 'sveltestrap';

 import SignUpForm from '$lib/components/SignUpForm.svelte';
 import KratosForm from '$lib/components/KratosForm.svelte';
 import { toasts } from '$lib/stores/toasts';
 import { t } from '$lib/translations';

 function next() {
     toasts.addToast({
         title: $t('account.signup.success'),
         message: $t('email.instruction.check-inbox'),
         type: 'success',
         timeout: 5000,
     });
     goto('/login');
 }
</script>

<Container class="my-3">
    <Row>
        <Col sm="4" class="d-none d-sm-flex flex-column justify-content-center">
            <img src="img/signup.webp" alt="Welcome aboard!">
        </Col>
        <Col sm="8" class="d-flex flex-column justify-content-center">
            <Card>
                <CardHeader>
                    <h6 class="card-title my-1 fw-bold">
                        {$t('account.signup.join-call')}
                    </h6>
                </CardHeader>
                <CardBody>
                    {#if window.happydomain_ory_kratos_url}
                        <KratosForm
                            flow="registration"
                            on:success={next}
                        />
                    {:else}
                        <SignUpForm
                            on:success={next}
                        />
                    {/if}
                </CardBody>
            </Card>
            <div class="mt-3 text-justify">
                Join now our open source and free (as freedom) DNS platform, to manage your domains easily!
            </div>
        </Col>
    </Row>
</Container>
