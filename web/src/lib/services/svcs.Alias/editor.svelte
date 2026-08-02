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
    import BasicInput from "$lib/components/inputs/basic.svelte";
    import { getProviderSpec } from "$lib/api/provider_specs";
    import type { dnsRR } from "$lib/dns_rr";
    import type { SvcsAliasBody } from "$lib/services_bodies";
    import type { Domain } from "$lib/model/domain";
    import { getAvailableResourceTypes, type ProviderInfos } from "$lib/model/provider";
    import { getRrtype, newRR, nsrrtype } from "$lib/dns_rr";
    import { providers_idx } from "$lib/stores/providers";

    interface Props {
        dn: string;
        origin: Domain;
        readonly?: boolean;
        value: SvcsAliasBody;
    }

    let { dn, origin, readonly = false, value = $bindable() }: Props = $props();

    // The Alias service holds the record itself, under a key the generated
    // dnsResource interface does not know about.
    let service = $derived(value as Record<string, any>);

    // The kinds of alias, from the most standard to the most provider specific.
    // Beside CNAME and DNAME, they are what DNSControl calls pseudo-types: they
    // have no DNS wire format, so only the providers declaring they handle them
    // accept them.
    const ALIAS_KINDS: Record<string, string> = {
        CNAME: "Points a name at another one. That name cannot hold any other record.",
        DNAME: "Points a whole subtree at another one.",
        ALIAS: "Behaves like a CNAME, but can live at the root of your domain, next to your other records. Your provider resolves it for you.",
        // No provider ever advertises ANAME: it is the spelling some of them
        // give to an ALIAS on their own side, so it is only ever read back. It
        // stays listed here so that such a record keeps a name and a
        // description in the editor.
        ANAME: "Behaves like a CNAME, but can live at the root of your domain. This is how your provider spells an ALIAS.",
        R53_ALIAS: "Points at an AWS resource, resolved by Route 53.",
        AZURE_ALIAS: "Points at an Azure resource, resolved by Azure DNS.",
        AKAMAICDN: "Points at an Akamai EdgeDNS resource.",
        AKAMAITLC: "Points at an Akamai traffic control resource.",
    };

    // Beside the target, some kinds carry the type of the resource they point
    // at. The lists are the ones DNSControl validates against.
    const ATYPE_CHOICES: Record<string, Array<string>> = {
        R53_ALIAS: [
            "A",
            "AAAA",
            "CNAME",
            "MX",
            "TXT",
            "PTR",
            "SRV",
            "SPF",
            "NAPTR",
            "CAA",
            "DS",
            "TLSA",
            "SSHFP",
            "SVCB",
            "HTTPS",
            "SOA",
        ],
        AZURE_ALIAS: ["A", "AAAA", "CNAME"],
    };

    // The answer type of an Akamai traffic control record: which address
    // families the load balancer is allowed to answer with.
    const ANSWER_TYPE_CHOICES = ["DUAL", "A", "AAAA"];

    // What the target of each kind actually is, as the standard types and the
    // Akamai ones point at a domain name where the two cloud ones point at a
    // resource of their provider.
    const TARGET_HINTS: Record<string, { description: string; placeholder: string }> = {
        R53_ALIAS: {
            description: "The AWS resource this alias points to",
            placeholder: "abcdef.cloudfront.net.",
        },
        AZURE_ALIAS: {
            description: "The identifier of the Azure resource this alias points to",
            placeholder:
                "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/my-group/providers/Microsoft.Network/dnszones/example.com/A/www",
        },
    };

    const DEFAULT_TARGET_HINT = {
        description: "The domain name this alias points to",
        placeholder: "example.com.",
    };

    // The target lives directly on the record for the standard types, and under
    // Data for the pseudo-types, which travel as a private use record.
    function getTarget(rr: dnsRR): string {
        const rdata = rr.Data as { Target?: string } | undefined;

        return (rdata ? rdata.Target : (rr.Target as string)) || "";
    }

    function setTarget(rr: dnsRR, target: string) {
        const rdata = rr.Data as { Target?: string } | undefined;

        if (rdata) {
            rdata.Target = target;
        } else {
            rr.Target = target;
        }
    }

    // Read one of the extra fields, which only exist on the rdata of the kinds
    // declaring them.
    function getField(rr: dnsRR, field: string): string {
        const rdata = rr.Data as Record<string, string> | undefined;

        return (rdata ? rdata[field] : undefined) || "";
    }

    // Write an extra field back, but only when the current kind knows it, so
    // that switching kinds does not leave a stray field behind.
    function setField(rr: dnsRR, field: string, v: string) {
        const rdata = rr.Data as Record<string, string> | undefined;

        if (rdata && field in rdata) {
            rdata[field] = v;
        }
    }

    // Initialize with a CNAME, the only kind every provider supports.
    if (!(value as Record<string, any>).record) {
        (value as Record<string, any>).record = newRR("", getRrtype("CNAME"));
    }

    let kind = $state(nsrrtype(((value as Record<string, any>).record as dnsRR).Hdr.Rrtype));
    let target = $state(getTarget((value as Record<string, any>).record as dnsRR));
    let atype = $state(getField((value as Record<string, any>).record as dnsRR, "AType"));
    let zoneid = $state(getField((value as Record<string, any>).record as dnsRR, "ZoneID"));
    let evaluateTargetHealth = $state(
        getField((value as Record<string, any>).record as dnsRR, "EvaluateTargetHealth") || "false",
    );
    let answerType = $state(
        getField((value as Record<string, any>).record as dnsRR, "AnswerType") || "DUAL",
    );

    let atypeChoices = $derived(ATYPE_CHOICES[kind]);
    let targetHint = $derived(TARGET_HINTS[kind] || DEFAULT_TARGET_HINT);

    $effect(() => {
        let record = service.record as dnsRR;

        if (record.Hdr.Rrtype !== getRrtype(kind)) {
            const changed = newRR(record.Hdr.Name, getRrtype(kind));
            changed.Hdr.Ttl = record.Hdr.Ttl;

            record = changed;
            service.record = changed;
        }

        // The kinds do not agree on the resource types they accept, so a type
        // kept from the previously selected kind may not be one of the choices
        // now offered. Fall back on the first of them rather than let the record
        // travel with a value its provider would reject.
        if (atypeChoices && atypeChoices.indexOf(atype) < 0) {
            atype = atypeChoices[0];
        }

        setTarget(record, target);
        setField(record, "AType", atype);
        setField(record, "ZoneID", zoneid);
        setField(record, "EvaluateTargetHealth", evaluateTargetHealth);
        setField(record, "AnswerType", answerType);
    });

    let provider_specs: ProviderInfos | null = $state(null);
    $effect(() => {
        const provider = origin.id_provider ? $providers_idx[origin.id_provider] : undefined;
        if (!provider) return;

        getProviderSpec(provider._srctype).then((prvdspecs) => {
            provider_specs = prvdspecs;
        });
    });

    // Only offer the kinds this provider declares it handles. The current one is
    // always kept, so that a record read from the zone stays editable even when
    // the provider stopped advertising its type.
    let availableKinds = $derived.by(() => {
        if (provider_specs === null) {
            return [kind];
        }

        const availableResourceTypes = getAvailableResourceTypes(provider_specs);

        return Object.keys(ALIAS_KINDS).filter(
            (k) => availableResourceTypes.indexOf(getRrtype(k)) >= 0 || k === kind,
        );
    });
</script>

<div>
    <BasicInput
        class="mt-3"
        edit
        index="kind"
        specs={{
            id: "kind",
            label: "Kind of alias",
            description: ALIAS_KINDS[kind],
            type: "string",
            choices: availableKinds,
        }}
        {readonly}
        bind:value={kind}
    />
    {#if atypeChoices}
        <BasicInput
            class="mt-3"
            edit
            index="atype"
            specs={{
                id: "atype",
                label: "Resource type",
                description:
                    "The kind of record the aliased resource answers with, as your provider needs to know it beforehand",
                type: "string",
                choices: atypeChoices,
            }}
            {readonly}
            bind:value={atype}
        />
    {/if}
    <BasicInput
        class="mt-3"
        edit
        index="target"
        specs={{
            id: "target",
            label: "Target",
            description: targetHint.description,
            type: "string",
            placeholder: targetHint.placeholder,
        }}
        {readonly}
        bind:value={target}
    />
    {#if kind === "AKAMAITLC"}
        <BasicInput
            class="mt-3"
            edit
            index="answertype"
            specs={{
                id: "answertype",
                label: "Answer type",
                description: "The addresses the Akamai load balancer is allowed to answer with",
                type: "string",
                choices: ANSWER_TYPE_CHOICES,
            }}
            {readonly}
            bind:value={answerType}
        />
    {/if}
    {#if kind === "R53_ALIAS"}
        <BasicInput
            class="mt-3"
            edit
            index="zoneid"
            specs={{
                id: "zoneid",
                label: "Hosted zone",
                description:
                    "The identifier of the Route 53 hosted zone holding the target. Leave it empty to let Route 53 find it out itself.",
                type: "string",
                placeholder: "Z2FDTNDATAQYW2",
            }}
            {readonly}
            bind:value={zoneid}
        />
        <BasicInput
            class="mt-3"
            edit
            index="evaluatetargethealth"
            specs={{
                id: "evaluatetargethealth",
                label: "Evaluate target health",
                description:
                    "Whether Route 53 stops answering with this alias when the target is reported unhealthy",
                type: "string",
                choices: ["false", "true"],
            }}
            {readonly}
            bind:value={evaluateTargetHealth}
        />
    {/if}
</div>
