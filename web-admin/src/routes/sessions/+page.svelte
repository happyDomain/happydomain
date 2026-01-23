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
        Col,
        Container,
        Icon,
        Row,
    } from "@sveltestrap/sveltestrap";

    import DeleteAllSessionsCard from './DeleteAllSessionsCard.svelte';
</script>

<Container class="flex-fill my-5">
    <Row class="mb-4">
        <Col md={8}>
            <h1 class="display-5">
                <Icon name="key"></Icon>
                Session Management
            </h1>
            <p class="lead text-muted">
                Manage all user sessions
            </p>
        </Col>
    </Row>

    <Row>
        <Col md={8} lg={6}>
            <DeleteAllSessionsCard />
        </Col>
    </Row>
</Container>
