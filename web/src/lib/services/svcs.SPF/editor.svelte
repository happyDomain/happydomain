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
    import { tick, untrack } from "svelte";

    import { Button, Icon, InputGroup, ListGroup, ListGroupItem } from "@sveltestrap/sveltestrap";

    import { fqdn } from "$lib/dns";
    import type { Domain } from "$lib/model/domain";
    import type { SvcsSPFBody } from "$lib/services_bodies";
    import BasicInput from "$lib/components/inputs/basic.svelte";
    import {
        issueDomId,
        severityIcon,
        type ComplianceIssue,
        type Severity,
        worstSeverity,
    } from "$lib/services/compliance";
    import type { dnsTypeTXT } from "$lib/dns_rr";
    import { newRR } from "$lib/dns_rr";
    import { t } from "$lib/translations";
    import { countLocalLookups, parseSPF, stringifySPF, termOrigins } from "./model";
    import { SPF_LOOKUP_MAX, walkSPF } from "./walk";

    interface Props {
        dn: string;
        origin: Domain;
        value: SvcsSPFBody;
        issues?: ComplianceIssue[];
    }

    let { dn, origin, value = $bindable(), issues = [] }: Props = $props();

    const DEFAULT_SPF = "v=spf1 -all";
    const ALL_RE = /^[-~?+]?all$/;

    if (!value["txt"]) {
        const txtRecord = newRR(
            untrack(() => dn),
            16,
        ) as dnsTypeTXT; // TXT record type is 16
        txtRecord.Txt = DEFAULT_SPF;
        value["txt"] = txtRecord;
    } else if (!/^\s*v=spf1(\s|$)/i.test(value["txt"].Txt || "")) {
        // The TXT slot may carry residue from another record type (DKIM, etc.)
        // when the service type was changed in place. Reset to a sane SPF
        // default rather than mis-parsing the foreign content.
        value["txt"].Txt = DEFAULT_SPF;
    }

    const initial = parseSPF(value["txt"].Txt || DEFAULT_SPF);
    let v = $state(initial.v ?? "spf1");
    let f = $state(initial.f);

    $effect(() => {
        // Force per-index dependency tracking so a `bind:value={f[i]}` write
        // re-runs this effect (and downstream EditorCompliance derivations).
        for (let i = 0; i < f.length; i++) void f[i];
        value["txt"]!.Txt = stringifySPF({ v, f });
    });

    let inputRefs: HTMLInputElement[] = $state([]);

    async function addDirective() {
        let newIdx: number;
        if (f.length >= 1 && ALL_RE.test(f[f.length - 1])) {
            newIdx = f.length - 1;
            f.splice(newIdx, 0, "");
        } else {
            newIdx = f.length;
            f.push("");
        }
        await tick();
        inputRefs[newIdx]?.focus();
    }

    function delDirective(idx: number) {
        f.splice(idx, 1);
    }

    // Same colors as the compliance messages, so a line reads as the one a
    // message is about. Warning text uses the emphasis shade for contrast.
    const severityText: Record<Severity, string> = {
        error: "text-danger",
        warning: "text-warning-emphasis",
        info: "text-info-emphasis",
    };

    // The lookup budget is shown whatever the count, so that it is known before
    // it runs out: the compliance check only speaks up past the limit.
    let lookups: { count: number; exceeded: boolean } | null = $state(null);

    $effect(() => {
        const val = { v, f: [...f] };
        const domain = fqdn(dn || "@", origin?.domain ?? "");
        if (v.toLowerCase() !== "spf1" || !domain) {
            lookups = null;
            return;
        }
        if (countLocalLookups(val).count === 0) {
            lookups = { count: 0, exceeded: false };
            return;
        }

        // Same delay as the compliance check, so that both share one walk.
        const ctrl = new AbortController();
        const timer = setTimeout(() => {
            walkSPF(domain, stringifySPF(val), ctrl.signal).then(
                (resp) => {
                    lookups = { count: resp.lookupCount ?? 0, exceeded: Boolean(resp.exceeded) };
                },
                () => {
                    // A count from a previous record would be misleading.
                    if (!ctrl.signal.aborted) lookups = null;
                },
            );
        }, 400);

        return () => {
            ctrl.abort();
            clearTimeout(timer);
        };
    });

    // Issues point at the terms of the record once stringified, where empty
    // lines are gone: bring each one back to the line it comes from.
    let issuesByLine = $derived.by(() => {
        const origins = termOrigins(f);
        const byLine: ComplianceIssue[][] = f.map(() => []);
        for (const issue of issues) {
            const term = /^f\[(\d+)\]$/.exec(issue.field ?? "");
            const line = term ? origins[Number(term[1])] : undefined;
            if (line !== undefined) byLine[line].push(issue);
        }
        return byLine;
    });

    function describedBy(lineIssues: ComplianceIssue[]): string | undefined {
        return lineIssues.length ? lineIssues.map(issueDomId).join(" ") : undefined;
    }
</script>

<div>
    <h4 class="text-primary pb-1 border-bottom border-1">Sender Policy Framework</h4>

    <BasicInput
        edit
        index="v"
        specs={{
            id: "v",
            label: "Version",
            placeholder: "spf1",
            type: "string",
            description: "Defines the version of SPF to use.",
        }}
        bind:value={v}
    />

    <h5 class="text-primary pb-1 border-bottom border-1">Directives</h5>
    <ListGroup>
        {#each f as _, i (i)}
            {@const lineIssues = issuesByLine[i] ?? []}
            {@const severity = worstSeverity(lineIssues)}
            <ListGroupItem class="p-0 {severity ? `directive-${severity}` : ''}">
                <InputGroup>
                    <input
                        class="form-control border-0"
                        aria-invalid={severity === "error" ? "true" : undefined}
                        aria-describedby={describedBy(lineIssues)}
                        bind:value={f[i]}
                        bind:this={inputRefs[i]}
                    />
                    {#if severity}
                        <span
                            class="input-group-text bg-transparent border-0 {severityText[
                                severity
                            ]}"
                            aria-hidden="true"
                        >
                            <Icon name={severityIcon[severity]} />
                        </span>
                    {/if}
                    <Button
                        type="button"
                        color="link"
                        class="text-danger"
                        onclick={() => delDirective(i)}
                    >
                        <Icon name="trash" />
                    </Button>
                </InputGroup>
            </ListGroupItem>
        {/each}
        <ListGroupItem tag="button" class="text-primary fw-bold" action onclick={addDirective}>
            <Icon name="plus-circle" />
            Add directive
        </ListGroupItem>
    </ListGroup>
    {#if lookups}
        <p class="small mt-2 mb-0 {lookups.exceeded ? 'text-danger' : 'text-muted'}">
            {$t("resources.SPF.lookups", { count: lookups.count, max: SPF_LOOKUP_MAX })}
        </p>
    {/if}
</div>

<style>
    /* The line takes the tint of the alert it is about. The bar is an inset
       shadow rather than a border, so that the line does not shift when an
       issue appears or goes away. */
    :global(.list-group-item.directive-error) {
        background-color: var(--bs-danger-bg-subtle);
        box-shadow: inset 4px 0 0 var(--bs-danger);
    }
    :global(.list-group-item.directive-warning) {
        background-color: var(--bs-warning-bg-subtle);
        box-shadow: inset 4px 0 0 var(--bs-warning);
    }
    :global(.list-group-item.directive-info) {
        background-color: var(--bs-info-bg-subtle);
        box-shadow: inset 4px 0 0 var(--bs-info);
    }
    /* The input would otherwise paint over both, focused or not. */
    :global(.list-group-item[class*="directive-"] .form-control) {
        background-color: transparent;
    }
</style>
