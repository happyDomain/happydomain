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
    import { page } from "$app/state";

    import { Col, Container, Row } from "@sveltestrap/sveltestrap";
</script>

<Container class="my-3">
    <Row>
        <Col xs="4" class="d-flex flex-column">
            <img src="/img/cat-error.webp" alt="The error cat" class="rounded rounded-lg" />
        </Col>
        <Col xs="8" class="d-flex flex-column pt-2">
            <div class="d-none d-sm-block my-4"></div>
            <div class="d-flex align-items-end">
                <h1>
                    Error {page.status}
                </h1>
                <hr class="ms-2 border-top border-primary border-3 flex-fill" />
            </div>
            {#if page.error}
                <p class="lead">
                    {page.error.message}
                </p>
            {/if}
            <hr class="mt-0 border-top border-secondary border-3" />
        </Col>
    </Row>
</Container>
