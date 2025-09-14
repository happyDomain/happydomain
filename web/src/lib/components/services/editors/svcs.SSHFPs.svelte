<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2025 happyDomain
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
    import { Button, Icon, Input } from "@sveltestrap/sveltestrap";
    import { t } from "$lib/translations";
    import type { Domain } from "$lib/model/domain";
    import type { dnsResource, dnsRR } from "$lib/dns_rr";

    interface Props {
        dn: string;
        origin: Domain;
        readonly?: boolean;
        value: dnsResource;
    }

    let { dn, origin, readonly = false, value = $bindable({}) }: Props = $props();
    const type = "svcs.SSHFPs";

    // Initialize sshfp array if needed - cast to array type
    if (!value["sshfp"]) {
        value["sshfp"] = [] as any;
    }

    // Type-safe accessor for sshfp records as array
    const getSSHFPArray = (): dnsRR[] => (value["sshfp"] as any) as dnsRR[];

    function addSSHFP() {
        const sshfpArray = getSSHFPArray();
        if (!sshfpArray) {
            value["sshfp"] = [] as any;
        }
        getSSHFPArray().push({ Algorithm: 1, Type: 1, FingerPrint: "" } as any);
    }

    function deleteSSHFP(idx: number) {
        const sshfpArray = getSSHFPArray();
        if (sshfpArray) {
            sshfpArray.splice(idx, 1);
        }
    }
</script>

<table class="table table-striped table-hover">
    <thead>
        <tr>
            <th>Algorithm</th>
            <th>Type</th>
            <th>Fingerprint</th>
            <th></th>
        </tr>
    </thead>
    <tbody>
        {#if getSSHFPArray() && getSSHFPArray().length}
            {@const sshfpArray = getSSHFPArray()}
            {#each sshfpArray as sshfp, idx}
                <tr>
                    <td>
                        <Input type="number" bsSize="sm" bind:value={sshfpArray[idx].Algorithm} />
                    </td>
                    <td>
                        <Input type="number" bsSize="sm" bind:value={sshfpArray[idx].Type} />
                    </td>
                    <td>
                        <Input bsSize="sm" bind:value={sshfpArray[idx].FingerPrint} />
                    </td>
                    <td>
                        <Button
                            type="button"
                            color="danger"
                            outline
                            size="sm"
                            onclick={() => deleteSSHFP(idx)}
                        >
                            <Icon name="trash" />
                        </Button>
                    </td>
                </tr>
            {/each}
        {/if}
    </tbody>
    <tfoot>
        <tr>
            <td colspan="4">
                <Button
                    type="button"
                    color="primary"
                    outline
                    size="sm"
                    onclick={addSSHFP}
                >
                    <Icon name="plus" />
                    {$t("common.new-row")}
                </Button>
            </td>
        </tr>
    </tfoot>
</table>
