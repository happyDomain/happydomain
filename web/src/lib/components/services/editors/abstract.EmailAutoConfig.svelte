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
    import { Alert, FormGroup, Input, Label } from "@sveltestrap/sveltestrap";

    import type { Domain } from "$lib/model/domain";
    import type { dnsTypeSRV, dnsTypeCNAME } from "$lib/dns_rr";
    import BasicInput from "$lib/components/inputs/basic.svelte";
    import { t } from "$lib/translations";

    interface Props {
        dn: string;
        origin: Domain;
        value: Record<string, any>;
    }

    let { dn, origin, value = $bindable({}) }: Props = $props();

    // ── Parsing helpers (record → form fields) ────────────────────────────

    // Pull the protocol identifier out of an SRV owner name like
    // "_imaps._tcp" or "_imaps._tcp.example.com.".
    function srvProtocol(srv: dnsTypeSRV | undefined | null): string {
        if (!srv?.Hdr?.Name) return "";
        const n = srv.Hdr.Name.replace(/^_/, "");
        const i = n.indexOf(".");
        return i >= 0 ? n.slice(0, i) : n;
    }
    function stripDot(s: string | undefined | null): string {
        return (s ?? "").replace(/\.$/, "");
    }
    function ensureTrailingDot(host: string): string {
        if (!host) return "";
        return host.endsWith(".") ? host : host + ".";
    }

    // ── Initial form state, derived from raw records ─────────────────────

    // The backend service-spec usecase auto-allocates pointer-to-DNS fields
    // with empty stub records (Hdr.Name == "") when serving a freshly-created
    // service. Drop those before reading anything else, otherwise the
    // unedited form would round-trip a phantom SRV/CNAME back to the zone.
    function isStubRecord(r: { Hdr?: { Name?: string } } | null | undefined): boolean {
        return r != null && (!r.Hdr || !r.Hdr.Name);
    }
    if (isStubRecord(value.incomingSRV)) value.incomingSRV = null;
    if (isStubRecord(value.outgoingSRV)) value.outgoingSRV = null;
    if (isStubRecord(value.autoconfigCNAME)) value.autoconfigCNAME = null;
    if (isStubRecord(value.autodiscoverCNAME)) value.autodiscoverCNAME = null;
    if (isStubRecord(value.autodiscoverSRV)) value.autodiscoverSRV = null;

    const incomingSRV = value.incomingSRV as dnsTypeSRV | undefined;
    const outgoingSRV = value.outgoingSRV as dnsTypeSRV | undefined;

    let incomingType = $state<string>(srvProtocol(incomingSRV) || "imaps");
    let incomingHost = $state<string>(stripDot(incomingSRV?.Target) || "");
    let incomingPort = $state<number>(incomingSRV?.Port ?? 993);

    let outgoingType = $state<string>(srvProtocol(outgoingSRV) || "submission");
    let outgoingHost = $state<string>(stripDot(outgoingSRV?.Target) || "");
    let outgoingPort = $state<number>(outgoingSRV?.Port ?? 587);

    let publishHTTP = $state<boolean>(
        value.autoconfigCNAME != null || value.autodiscoverCNAME != null,
    );

    if (value.usernameFormat === undefined) value.usernameFormat = "%EMAILADDRESS%";

    // Snapshot of initial form state — used to skip rebuilding records when
    // the user hasn't actually edited anything (preserves analyzed records
    // verbatim).
    const initialIncoming = JSON.stringify({
        type: srvProtocol(incomingSRV),
        host: stripDot(incomingSRV?.Target),
        port: incomingSRV?.Port,
    });
    const initialOutgoing = JSON.stringify({
        type: srvProtocol(outgoingSRV),
        host: stripDot(outgoingSRV?.Target),
        port: outgoingSRV?.Port,
    });
    const initialPublishHTTP = publishHTTP;

    // ── Defaults wiring ──────────────────────────────────────────────────

    const incomingDefaults: Record<string, number> = {
        imap: 143,
        imaps: 993,
        pop3: 110,
        pop3s: 995,
    };
    const outgoingDefaults: Record<string, number> = {
        submission: 587,
        submissions: 465,
    };

    let prevIncomingType = incomingType;
    let prevOutgoingType = outgoingType;

    $effect(() => {
        if (incomingType !== prevIncomingType) {
            const oldDefault = incomingDefaults[prevIncomingType];
            if (!incomingPort || incomingPort === oldDefault) {
                incomingPort = incomingDefaults[incomingType];
            }
            prevIncomingType = incomingType;
        }
    });
    $effect(() => {
        if (outgoingType !== prevOutgoingType) {
            const oldDefault = outgoingDefaults[prevOutgoingType];
            if (!outgoingPort || outgoingPort === oldDefault) {
                outgoingPort = outgoingDefaults[outgoingType];
            }
            prevOutgoingType = outgoingType;
        }
    });

    // ── Record reconstruction (form fields → records) ────────────────────

    // Preserve the analyzed records' TTL/priority/weight when the user only
    // changes display fields. Captured once at script init so the rebuild
    // effects don't re-read reactive `value.*` (which would loop).
    const baseIncomingTtl = incomingSRV?.Hdr?.Ttl ?? 0;
    const baseIncomingPriority = incomingSRV?.Priority ?? 0;
    const baseIncomingWeight = incomingSRV?.Weight ?? 1;
    const baseOutgoingTtl = outgoingSRV?.Hdr?.Ttl ?? 0;
    const baseOutgoingPriority = outgoingSRV?.Priority ?? 0;
    const baseOutgoingWeight = outgoingSRV?.Weight ?? 1;

    function makeSRV(
        name: string,
        port: number,
        target: string,
        ttl: number,
        priority: number,
        weight: number,
    ): dnsTypeSRV {
        return {
            Hdr: { Name: name, Rrtype: 33, Class: 1, Ttl: ttl, Rdlength: 0 },
            Priority: priority,
            Weight: weight,
            Port: port,
            Target: target,
        };
    }
    function makeCNAME(name: string, target: string): dnsTypeCNAME {
        return {
            Hdr: { Name: name, Rrtype: 5, Class: 1, Ttl: 0, Rdlength: 0 },
            Target: target,
        };
    }

    // Incoming SRV — only rebuild when user touched the inputs.
    $effect(() => {
        const cur = JSON.stringify({ type: incomingType, host: incomingHost, port: incomingPort });
        if (cur === initialIncoming) return;
        value.incomingSRV = incomingHost
            ? makeSRV(
                  `_${incomingType}._tcp`,
                  incomingPort,
                  ensureTrailingDot(incomingHost),
                  baseIncomingTtl,
                  baseIncomingPriority,
                  baseIncomingWeight,
              )
            : null;
    });

    $effect(() => {
        const cur = JSON.stringify({ type: outgoingType, host: outgoingHost, port: outgoingPort });
        if (cur === initialOutgoing) return;
        value.outgoingSRV = outgoingHost
            ? makeSRV(
                  `_${outgoingType}._tcp`,
                  outgoingPort,
                  ensureTrailingDot(outgoingHost),
                  baseOutgoingTtl,
                  baseOutgoingPriority,
                  baseOutgoingWeight,
              )
            : null;
    });

    // HTTP discovery — toggles add/remove the three records. The CNAME/SRV
    // targets point at the happyDomain instance currently serving this UI,
    // which is also the host that answers Mozilla Autoconfig and Microsoft
    // Autodiscover XML.
    const discoveryTarget = ensureTrailingDot(
        typeof window !== "undefined" ? window.location.hostname : "",
    );
    $effect(() => {
        if (publishHTTP === initialPublishHTTP) return;
        if (publishHTTP) {
            value.autoconfigCNAME = makeCNAME("autoconfig", discoveryTarget);
            value.autodiscoverCNAME = makeCNAME("autodiscover", discoveryTarget);
            value.autodiscoverSRV = makeSRV(
                "_autodiscover._tcp",
                443,
                discoveryTarget,
                0,
                0,
                0,
            );
        } else {
            value.autoconfigCNAME = null;
            value.autodiscoverCNAME = null;
            value.autodiscoverSRV = null;
        }
    });

    // ── UI metadata ──────────────────────────────────────────────────────

    const authChoices = [
        { value: "password-cleartext", label: "Password (cleartext, over TLS)" },
        { value: "password-encrypted", label: "Password (encrypted)" },
        { value: "OAuth2", label: "OAuth2" },
        { value: "NTLM", label: "NTLM" },
    ];

    const incomingProtocols = [
        { value: "imaps", label: "IMAPS (port 993, TLS)" },
        { value: "imap", label: "IMAP (port 143, STARTTLS or plain)" },
        { value: "pop3s", label: "POP3S (port 995, TLS)" },
        { value: "pop3", label: "POP3 (port 110, STARTTLS or plain)" },
    ];

    const outgoingProtocols = [
        { value: "submission", label: "Submission (port 587, STARTTLS)" },
        { value: "submissions", label: "Submissions (port 465, TLS)" },
    ];

    const usernameFormats = [
        { value: "%EMAILADDRESS%", label: "Full email address (user@example.com)" },
        { value: "%EMAILLOCALPART%", label: "Local part only (user)" },
    ];

    let portWarning = $derived.by(() => {
        const issues: string[] = [];
        if (incomingPort && (incomingPort < 1 || incomingPort > 65535))
            issues.push("Incoming port must be between 1 and 65535");
        if (outgoingPort && (outgoingPort < 1 || outgoingPort > 65535))
            issues.push("Outgoing port must be between 1 and 65535");
        const inDef = incomingDefaults[incomingType];
        const outDef = outgoingDefaults[outgoingType];
        if (incomingPort && inDef && incomingPort !== inDef)
            issues.push(`Non-standard port ${incomingPort} for ${incomingType}`);
        if (outgoingPort && outDef && outgoingPort !== outDef)
            issues.push(`Non-standard port ${outgoingPort} for ${outgoingType}`);
        return issues;
    });
</script>

<div>
    <h4 class="text-primary pb-1 border-bottom border-1">
        {$t("services.email-autoconfig.title", { default: "Email Auto-configuration" })}
    </h4>
    <p class="text-muted small">
        {$t("services.email-autoconfig.intro", {
            default:
                "Publishes IMAP/POP/SMTP settings via RFC 6186 SRV records, Mozilla Autoconfig, and Microsoft Autodiscover so mail clients can configure themselves automatically.",
        })}
    </p>

    {#if portWarning.length > 0}
        <Alert color="warning" class="py-2 small mb-3">
            {#each portWarning as w}
                <div>{w}</div>
            {/each}
        </Alert>
    {/if}

    <h5 class="mt-3 text-primary pb-1 border-bottom border-1">
        {$t("services.email-autoconfig.incoming", { default: "Incoming server" })}
    </h5>

    <FormGroup row>
        <Label md="4" class="text-md-end text-primary">{$t("services.email-autoconfig.protocol", { default: "Protocol" })}</Label>
        <div class="col-md-8">
            <Input type="select" bind:value={incomingType} bsSize="sm">
                {#each incomingProtocols as p}
                    <option value={p.value}>{p.label}</option>
                {/each}
            </Input>
        </div>
    </FormGroup>

    <BasicInput
        edit
        index="incomingHost"
        specs={{
            id: "incomingHost",
            label: $t("services.email-autoconfig.hostname", { default: "Hostname" }),
            placeholder: "imap.example.com",
            type: "string",
            required: true,
            description: $t("services.email-autoconfig.incoming-hostname-desc", {
                default: "FQDN of your IMAP/POP3 server.",
            }),
        }}
        bind:value={incomingHost}
    />

    <BasicInput
        edit
        index="incomingPort"
        specs={{
            id: "incomingPort",
            label: $t("services.email-autoconfig.port", { default: "Port" }),
            placeholder: "993",
            type: "uint16",
            required: true,
        }}
        bind:value={incomingPort}
    />

    <FormGroup row>
        <Label md="4" class="text-md-end text-primary">{$t("services.email-autoconfig.auth", { default: "Authentication" })}</Label>
        <div class="col-md-8">
            <Input type="select" bind:value={value.incomingAuth} bsSize="sm">
                {#each authChoices as a}
                    <option value={a.value}>{a.label}</option>
                {/each}
            </Input>
        </div>
    </FormGroup>

    <h5 class="mt-3 text-primary pb-1 border-bottom border-1">
        {$t("services.email-autoconfig.outgoing", { default: "Outgoing server" })}
    </h5>

    <FormGroup row>
        <Label md="4" class="text-md-end text-primary">{$t("services.email-autoconfig.protocol", { default: "Protocol" })}</Label>
        <div class="col-md-8">
            <Input type="select" bind:value={outgoingType} bsSize="sm">
                {#each outgoingProtocols as p}
                    <option value={p.value}>{p.label}</option>
                {/each}
            </Input>
        </div>
    </FormGroup>

    <BasicInput
        edit
        index="outgoingHost"
        specs={{
            id: "outgoingHost",
            label: $t("services.email-autoconfig.hostname", { default: "Hostname" }),
            placeholder: "smtp.example.com",
            type: "string",
            required: true,
            description: $t("services.email-autoconfig.outgoing-hostname-desc", {
                default: "FQDN of your SMTP submission server.",
            }),
        }}
        bind:value={outgoingHost}
    />

    <BasicInput
        edit
        index="outgoingPort"
        specs={{
            id: "outgoingPort",
            label: $t("services.email-autoconfig.port", { default: "Port" }),
            placeholder: "587",
            type: "uint16",
            required: true,
        }}
        bind:value={outgoingPort}
    />

    <FormGroup row>
        <Label md="4" class="text-md-end text-primary">{$t("services.email-autoconfig.auth", { default: "Authentication" })}</Label>
        <div class="col-md-8">
            <Input type="select" bind:value={value.outgoingAuth} bsSize="sm">
                {#each authChoices as a}
                    <option value={a.value}>{a.label}</option>
                {/each}
            </Input>
        </div>
    </FormGroup>

    <h5 class="mt-3 text-primary pb-1 border-bottom border-1">
        {$t("services.email-autoconfig.discovery", { default: "Discovery" })}
    </h5>

    <FormGroup>
        <Input
            type="checkbox"
            label={$t("services.email-autoconfig.publish-http", {
                default: "Publish HTTP discovery (autoconfig./autodiscover. CNAMEs)",
            })}
            bind:checked={publishHTTP}
        />
        <p class="small text-muted mt-1 mb-0">
            {$t("services.email-autoconfig.publish-http-desc", {
                default:
                    "When enabled, happyDomain creates CNAMEs for autoconfig.<your-domain> and autodiscover.<your-domain> pointing to this happyDomain instance, which serves the corresponding XML over HTTPS so Thunderbird and Outlook can self-configure.",
            })}
        </p>
    </FormGroup>

    <BasicInput
        edit
        index="exchangeServer"
        specs={{
            id: "exchangeServer",
            label: $t("services.email-autoconfig.exchange", { default: "Exchange Server (optional)" }),
            placeholder: "mail.example.com",
            type: "string",
            description: $t("services.email-autoconfig.exchange-desc", {
                default:
                    "Hostname of an on-premises Microsoft Exchange server. Enables MAPI/EWS in the Autodiscover response.",
            }),
        }}
        bind:value={value.exchangeServer}
    />

    <h5 class="mt-3 text-primary pb-1 border-bottom border-1">
        {$t("services.email-autoconfig.branding", { default: "Branding" })}
    </h5>

    <BasicInput
        edit
        index="displayName"
        specs={{
            id: "displayName",
            label: $t("services.email-autoconfig.display-name", { default: "Provider Display Name" }),
            placeholder: "Example Mail",
            type: "string",
        }}
        bind:value={value.displayName}
    />

    <BasicInput
        edit
        index="displayShortName"
        specs={{
            id: "displayShortName",
            label: $t("services.email-autoconfig.display-short-name", { default: "Short Name" }),
            placeholder: "Example",
            type: "string",
        }}
        bind:value={value.displayShortName}
    />

    <FormGroup row>
        <Label md="4" class="text-md-end text-primary">{$t("services.email-autoconfig.username-format", { default: "Username Format" })}</Label>
        <div class="col-md-8">
            <Input type="select" bind:value={value.usernameFormat} bsSize="sm">
                {#each usernameFormats as f}
                    <option value={f.value}>{f.label}</option>
                {/each}
            </Input>
        </div>
    </FormGroup>
</div>
