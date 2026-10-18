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
    import { Badge, Card, CardBody, CardHeader, Icon } from "@sveltestrap/sveltestrap";

    interface Props {
        policy: string;
        hasKeyset: boolean;
        class?: string;
    }

    let { policy, hasKeyset, class: className = "" }: Props = $props();
</script>

<Card class={className}>
    <CardHeader>
        <Icon name="gear-fill"></Icon>
        Policy
    </CardHeader>
    <CardBody>
        <div class="d-flex gap-4 align-items-center mb-3">
            <div>
                <small class="text-muted d-block">New credentials are stored</small>
                {#if policy === "instance"}
                    <Badge color="success"><Icon name="lock-fill"></Icon> instance</Badge>
                {:else}
                    <Badge color="secondary"
                        ><Icon name="unlock-fill"></Icon> {policy || "unknown"}</Badge
                    >
                {/if}
            </div>
            <div>
                <small class="text-muted d-block">Instance keyset</small>
                {#if hasKeyset}
                    <Badge color="success">configured</Badge>
                {:else}
                    <Badge color="secondary">not configured</Badge>
                {/if}
            </div>
        </div>
        <p class="text-muted mb-0">
            {#if policy === "instance"}
                Every credential saved is encrypted under the instance keyset, held outside the
                database.
            {:else}
                Credentials saved are stored in clear in the database.
            {/if}
            The policy is set with <code>secret-policy</code>, the keyset with
            <code>secret-keyset-file</code>; changing either needs a restart.
        </p>
    </CardBody>
</Card>
