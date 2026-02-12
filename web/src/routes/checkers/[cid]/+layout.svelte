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
    import { Col, Container, Row } from "@sveltestrap/sveltestrap";
    import { page } from "$app/state";

    import CheckerSidebar from "$lib/components/checkers/CheckerSidebar.svelte";

    let {
        children,
    }: {
        children?: import("svelte").Snippet;
    } = $props();

    let cid = $derived(page.params.cid!);
</script>

<Container fluid class="d-flex flex-column flex-fill">
    <Row class="flex-fill">
        <Col
            sm={4}
            md={3}
            class="py-3 sticky-top d-flex flex-column"
            style="background-color: #edf5f2; overflow-y: auto; max-height: 100vh; z-index: 0"
        >
            <CheckerSidebar currentCheckId={cid} />
        </Col>
        <Col sm={8} md={9} class="d-flex flex-column">
            {@render children?.()}
        </Col>
    </Row>
</Container>
