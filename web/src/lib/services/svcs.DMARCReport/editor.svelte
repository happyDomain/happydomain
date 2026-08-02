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

    import type { Domain } from "$lib/model/domain";
    import type { dnsResource, dnsTypeTXT } from "$lib/dns_rr";
    import { getRrtype, newRR } from "$lib/dns_rr";
    import { t } from "$lib/translations";

    interface Props {
        dn: string;
        origin: Domain;
        value: dnsResource;
    }

    let { dn, origin, value = $bindable({}) }: Props = $props();

    const type = "svcs.DMARCReport";
    const SUFFIX = "._report._dmarc";

    if (!Array.isArray(value["txt"])) value["txt"] = [];

    function getDomain(rr: dnsTypeTXT): string {
        const n = rr.Hdr?.Name ?? "";
        return n.endsWith(SUFFIX) ? n.slice(0, -SUFFIX.length) : n;
    }

    function setDomain(idx: number, d: string) {
        const rrs = value["txt"] as dnsTypeTXT[];
        rrs[idx].Hdr.Name = (d || "") + SUFFIX;
    }

    function addDomain() {
        const rec = newRR("" + SUFFIX, getRrtype("TXT")) as dnsTypeTXT;
        rec.Txt = "v=DMARC1";
        (value["txt"] as dnsTypeTXT[]).push(rec);
    }

    function removeDomain(idx: number) {
        (value["txt"] as dnsTypeTXT[]).splice(idx, 1);
    }
</script>

<div>
    <h4 class="text-primary pb-1 border-bottom border-1">
        Domains allowed to send DMARC reports here
    </h4>
    <table class="table table-striped table-hover">
        <thead>
            <tr>
                <th>Domain</th>
                <th></th>
            </tr>
        </thead>
        <tbody>
            {#if (value["txt"] as dnsTypeTXT[]).length}
                {#each value["txt"] as dnsTypeTXT[] as rr, idx}
                    <tr>
                        <td>
                            <Input
                                bsSize="sm"
                                value={getDomain(rr)}
                                oninput={(e) =>
                                    setDomain(idx, (e.target as HTMLInputElement).value)}
                            />
                        </td>
                        <td>
                            <Button
                                type="button"
                                color="danger"
                                outline
                                size="sm"
                                onclick={() => removeDomain(idx)}
                            >
                                <Icon name="trash" />
                            </Button>
                        </td>
                    </tr>
                {/each}
            {:else}
                <tr>
                    <td colspan={2} class="fst-italic text-center">
                        {$t("common.no-content")}
                    </td>
                </tr>
            {/if}
        </tbody>
        <tfoot>
            <tr>
                <td colspan={2}>
                    <Button
                        type="button"
                        color="primary"
                        outline
                        size="sm"
                        onclick={addDomain}
                    >
                        <Icon name="plus" />
                        {$t("common.new-row")}
                    </Button>
                </td>
            </tr>
        </tfoot>
    </table>
</div>
