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
    import { untrack } from "svelte";

    import { Button, FormGroup, Input, Label } from "@sveltestrap/sveltestrap";

    import type { Domain } from "$lib/model/domain";
    import type { AbstractMTASTSBody } from "$lib/services_bodies";
    import type { dnsTypeCNAME, dnsTypeTXT } from "$lib/dns_rr";
    import BasicInput from "$lib/components/inputs/basic.svelte";
    import { appConfig } from "$lib/stores/config";
    import { buildContext } from "$lib/services/compliance";
    import { thisZone } from "$lib/stores/thiszone";
    import { t } from "$lib/translations";
    import { getZoneApexMxHosts, policyURL, stripTrailingDot } from "$lib/services/mta_sts";
    import { ensureTrailingDot, isStubRecord } from "$lib/services/records";
    import {
        DEFAULT_MAX_AGE,
        newPolicyId,
        policyFingerprint,
        renderPolicy,
        stringifyMTASTS,
    } from "./model";

    interface Props {
        dn: string;
        origin: Domain;
        value: AbstractMTASTSBody;
    }

    let { dn, origin, value = $bindable() }: Props = $props();

    // ── Stub cleanup ─────────────────────────────────────────────────────

    // Drop the stub records the backend allocates, otherwise the unedited form
    // would round-trip a phantom TXT/CNAME back to the zone.
    if (isStubRecord(value.txt)) value.txt = undefined;
    if (isStubRecord(value.policyCNAME)) value.policyCNAME = undefined;

    // ── Defaults ─────────────────────────────────────────────────────────

    if (!value.mode) value.mode = "testing";
    if (!value.maxAge) value.maxAge = DEFAULT_MAX_AGE;
    if (!value.mx) value.mx = [];

    // A brand new service starts with the MX records the zone already
    // publishes: that is what the policy is meant to describe, and retyping
    // them by hand is the easiest way to lock mail out of the domain.
    // The policy file lists bare host names, never FQDNs with a final dot.
    const zoneMx = untrack(() =>
        getZoneApexMxHosts(buildContext(dn, origin, $thisZone)).map(stripTrailingDot),
    );
    if (value.mx.length === 0 && zoneMx.length > 0) value.mx = zoneMx;

    // ── Hosting toggle ───────────────────────────────────────────────────

    // value.policyCNAME is the single source of truth; the checkbox state is
    // derived from it rather than mirrored into a second variable.
    let hostPolicy = $derived(value.policyCNAME != null);

    // The CNAME points at the FQDN the backend analyzer recognizes as this
    // happyDomain instance (--service-hosting-host), which is not
    // necessarily the hostname the browser used to reach the UI.
    const policyTarget = ensureTrailingDot($appConfig.service_hosting_host ?? "");

    function toggleHostPolicy(checked: boolean) {
        value.policyCNAME = checked
            ? ({
                  Hdr: { Name: "mta-sts", Rrtype: 5, Class: 1, Ttl: 0, Rdlength: 0 },
                  Target: policyTarget,
              } as dnsTypeCNAME)
            : undefined;
    }

    // ── Policy id ────────────────────────────────────────────────────────

    // RFC 8461 sec. 3.1: senders re-fetch the policy when the id changes, so
    // the id has to move whenever the policy does. Merely opening the form
    // must not touch it, hence the fingerprint captured before any edit.
    const initialPolicy = untrack(() => policyFingerprint(value));

    // The record this effect writes is read back through untrack(): it is an
    // output, not a dependency, and tracking it would make the effect retrigger
    // on its own assignment.
    $effect(() => {
        const current = policyFingerprint(value);
        const existing = untrack(() => value.txt as dnsTypeTXT | undefined);

        if (current === initialPolicy && existing) return;

        const txt = stringifyMTASTS({ v: "STSv1", id: newPolicyId() }, existing?.Txt ?? "");

        if (existing?.Txt === txt) return;

        value.txt = {
            Hdr: {
                Name: "_mta-sts",
                Rrtype: 16,
                Class: 1,
                Ttl: existing?.Hdr?.Ttl ?? 0,
                Rdlength: 0,
            },
            Txt: txt,
        } as dnsTypeTXT;
    });

    // ── MX list editing ──────────────────────────────────────────────────

    function addMX() {
        value.mx = [...(value.mx ?? []), ""];
    }
    function removeMX(index: number) {
        value.mx = (value.mx ?? []).filter((_, i) => i !== index);
    }

    let preview = $derived(renderPolicy(value));
    let policyUrl = $derived(policyURL(origin.domain));

    const modes = [
        {
            value: "testing",
            label: $t("services.mta-sts.mode-testing", {
                default: "Testing — report failures, still deliver",
            }),
        },
        {
            value: "enforce",
            label: $t("services.mta-sts.mode-enforce", {
                default: "Enforce — refuse to deliver when TLS fails",
            }),
        },
        {
            value: "none",
            label: $t("services.mta-sts.mode-none", { default: "None — withdraw the policy" }),
        },
    ];
</script>

<div>
    <h4 class="text-primary pb-1 border-bottom border-1">
        {$t("services.mta-sts.title", { default: "MTA Strict Transport Security" })}
    </h4>
    <p class="text-muted small">
        {$t("services.mta-sts.intro", {
            default:
                "Tells other mail servers to only deliver to your domain over authenticated TLS. Beyond the DNS record, the standard needs a policy file served over HTTPS: happyDomain can host it for you.",
        })}
    </p>

    <h5 class="mt-3 text-primary pb-1 border-bottom border-1">
        {$t("services.mta-sts.policy", { default: "Policy" })}
    </h5>

    <FormGroup row>
        <Label md="4" class="text-md-end text-primary"
            >{$t("services.mta-sts.mode", { default: "Mode" })}</Label
        >
        <div class="col-md-8">
            <Input type="select" bind:value={value.mode} bsSize="sm">
                {#each modes as m (m.value)}
                    <option value={m.value}>{m.label}</option>
                {/each}
            </Input>
        </div>
    </FormGroup>

    <BasicInput
        edit
        index="maxAge"
        specs={{
            id: "maxAge",
            label: $t("services.mta-sts.max-age", { default: "Max Age" }),
            placeholder: "604800",
            type: "uint32",
            required: true,
            description: $t("services.mta-sts.max-age-desc", {
                default:
                    "How long (in seconds) senders cache this policy. One week is the recommended minimum; up to one year is allowed.",
            }),
        }}
        bind:value={value.maxAge}
    />

    <FormGroup row>
        <Label md="4" class="text-md-end text-primary"
            >{$t("services.mta-sts.mx", { default: "Authorized MX" })}</Label
        >
        <div class="col-md-8">
            <!-- Keyed by index: the entries are edited in place, so identity
                 has to follow the slot, not the (changing) value. -->
            {#each value.mx ?? [] as _, index (index)}
                <div class="d-flex gap-2 mb-2">
                    <Input
                        type="text"
                        bsSize="sm"
                        placeholder="mail.example.com"
                        bind:value={value.mx![index]}
                    />
                    <Button
                        size="sm"
                        color="danger"
                        outline
                        onclick={() => removeMX(index)}
                        title={$t("common.remove", { default: "Remove" })}
                    >
                        &times;
                    </Button>
                </div>
            {/each}
            <Button size="sm" color="secondary" outline onclick={addMX}>
                {$t("services.mta-sts.add-mx", { default: "Add an MX host" })}
            </Button>
            <p class="small text-muted mt-2 mb-0">
                {$t("services.mta-sts.mx-desc", {
                    default:
                        "Host names allowed to receive mail for this domain, as found in your MX records. A leading *. matches exactly one label.",
                })}
            </p>
        </div>
    </FormGroup>

    <h5 class="mt-3 text-primary pb-1 border-bottom border-1">
        {$t("services.mta-sts.hosting", { default: "Policy hosting" })}
    </h5>

    <FormGroup>
        <Input
            type="checkbox"
            label={$t("services.mta-sts.host-policy", {
                default: "Let happyDomain serve the policy file (mta-sts. CNAME)",
            })}
            checked={hostPolicy}
            on:change={(e) => toggleHostPolicy(e.currentTarget.checked)}
        />
        <p class="small text-muted mt-1 mb-0">
            {$t("services.mta-sts.host-policy-desc", {
                default:
                    "When enabled, happyDomain creates a CNAME for mta-sts.<your-domain> pointing to this happyDomain instance, which serves the policy below over HTTPS. Leave it off if you serve the file from your own web server.",
            })}
        </p>
    </FormGroup>

    <h5 class="mt-3 text-primary pb-1 border-bottom border-1">
        {$t("services.mta-sts.preview", { default: "Policy file" })}
    </h5>

    <p class="small text-muted mb-1">
        <code>{policyUrl}</code>
    </p>
    {#if preview}
        <pre class="bg-body-secondary border rounded p-2 small mb-0"><code>{preview}</code></pre>
    {:else}
        <p class="small text-muted fst-italic mb-0">
            {$t("services.mta-sts.preview-empty", {
                default:
                    "Nothing will be served yet: testing and enforce policies need at least one authorized MX.",
            })}
        </p>
    {/if}
</div>
