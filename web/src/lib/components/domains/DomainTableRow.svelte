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
    import { Badge, Button, ButtonGroup, Icon } from "@sveltestrap/sveltestrap";

    import ProviderLink from "$lib/components/providers/ProviderLink.svelte";
    import type { HappydnsDomainWithCheckStatus } from "$lib/api-base/types.gen";
    import { navigate } from "$lib/stores/config";
    import { t } from "$lib/translations";
    import { getStatusColor, getStatusIcon } from "$lib/utils/checkers";

    interface Props {
        domain: HappydnsDomainWithCheckStatus;
        ondelete: (event: Event, domain: HappydnsDomainWithCheckStatus) => void;
    }

    let { domain, ondelete }: Props = $props();
</script>

<tr
    style="cursor: pointer"
    onclick={() => navigate("/domains/" + encodeURIComponent(domain.domain))}
>
    <td class="fw-semibold">{domain.domain}</td>
    <td>{domain.group || ""}</td>
    <td>
        <ProviderLink id_provider={domain.id_provider} onclick={(e) => e.stopPropagation()} />
    </td>
    <td>
        <a
            href="/domains/{encodeURIComponent(domain.domain)}/checks"
            class="text-decoration-none"
            onclick={(e) => e.stopPropagation()}
        >
            <Badge color={getStatusColor(domain.last_check_status)}>
                <Icon name={getStatusIcon(domain.last_check_status)} />
            </Badge>
        </a>
    </td>
    <td class="text-end">
        <ButtonGroup size="sm">
            <Button
                color="outline-secondary"
                title={$t("domains.actions.view")}
                onclick={(e) => {
                    e.stopPropagation();
                    navigate("/domains/" + encodeURIComponent(domain.domain));
                }}
            >
                <Icon name="eye" />
            </Button>
            <Button
                color="outline-danger"
                title={$t("domains.stop")}
                onclick={(e) => ondelete(e, domain)}
            >
                <Icon name="trash" />
            </Button>
        </ButtonGroup>
    </td>
</tr>
