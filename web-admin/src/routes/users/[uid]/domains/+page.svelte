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
    import { page } from "$app/state";
    import { Button, Col, Container, Icon, Row } from "@sveltestrap/sveltestrap";
    import { getUsersByUidDomains } from "$lib/api-admin";
    import UserDomainsCard from "./UserDomainsCard.svelte";

    let userId = $derived(page.params.uid!);
    let domainsQ = $derived(getUsersByUidDomains({ path: { uid: userId } }));
</script>

<Container class="flex-fill my-5">
    <Row class="mb-4">
        <Col>
            <div class="d-flex align-items-center gap-1">
                <Button color="link" href="/users/{userId}" class="text-black">
                    <Icon name="chevron-left"></Icon>
                </Button>
                <h1 class="display-5 mb-0">User Domains</h1>
            </div>
        </Col>
    </Row>

    <UserDomainsCard {domainsQ} {userId} />
</Container>
