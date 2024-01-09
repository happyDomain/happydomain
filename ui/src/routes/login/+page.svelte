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
 import {
     Card,
     CardBody,
     CardHeader,
     Container,
     Col,
     Row,
 } from '@sveltestrap/sveltestrap';

 import { t } from '$lib/translations';
 import LoginForm from '$lib/components/LoginForm.svelte';
</script>

<Container class="my-3">
    <Row>
        <Col sm="4" class="d-none d-sm-flex flex-column">
          <img src="/img/login.webp" alt="Welcome back!">
        </Col>
        <Col sm="8" class="d-flex flex-column justify-content-center">
            <Card>
                <CardHeader>
                    <h6 class="card-title my-1 fw-bold">
                        {$t('account.signup.join-call')}
                    </h6>
                </CardHeader>
                <CardBody>
                    <LoginForm />
                </CardBody>
            </Card>
            <div class="text-center mt-4">
                {$t('account.ask-have')}
                <a href="/join" class="fw-bold">
                    {$t('account.join')}
                </a>
            </div>
        </Col>
    </Row>
</Container>
