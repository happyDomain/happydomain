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
    import { Button, FormGroup, Label } from "@sveltestrap/sveltestrap";

    import type { Domain } from "$lib/model/domain";
    import type { AbstractBIMIBody } from "$lib/services_bodies";
    import type { dnsTypeCNAME, dnsTypeTXT } from "$lib/dns_rr";
    import BasicInput from "$lib/components/inputs/basic.svelte";
    import { appConfig } from "$lib/stores/config";
    import { t } from "$lib/translations";
    import { ensureTrailingDot, isStubRecord } from "$lib/services/records";
    import {
        HOST_CNAME_NAME,
        RECORD_SUFFIX,
        certificateProblems,
        contentName,
        countCertificates,
        decodeBase64,
        encodeBase64,
        generatedRecord,
        logoProblems,
        selectorOf,
        serviceDomain,
    } from "./model";

    interface Props {
        dn: string;
        origin: Domain;
        value: AbstractBIMIBody;
    }

    let { dn, origin, value = $bindable() }: Props = $props();

    // Drop the stub records the backend allocates, then lay down the two
    // records this service always publishes: the TXT content is generated
    // from the files, and the CNAME is what makes them reachable.
    if (isStubRecord(value.txt)) value.txt = undefined;
    if (isStubRecord(value.hostCNAME)) value.hostCNAME = undefined;

    if (!value.txt) {
        value.txt = {
            Hdr: { Name: "default" + RECORD_SUFFIX, Rrtype: 16, Class: 1, Ttl: 0, Rdlength: 0 },
            Txt: "",
        } as dnsTypeTXT;
    }
    if (!value.hostCNAME) {
        // The target is the host the backend analyzer recognizes as ours, not
        // the browser's.
        value.hostCNAME = {
            Hdr: { Name: HOST_CNAME_NAME, Rrtype: 5, Class: 1, Ttl: 0, Rdlength: 0 },
            Target: ensureTrailingDot($appConfig.service_hosting_host ?? ""),
        } as dnsTypeCNAME;
    }

    let selector = $state(selectorOf(value.txt.Hdr?.Name) || "default");
    $effect(() => {
        if (value.txt?.Hdr) value.txt.Hdr.Name = selector + RECORD_SUFFIX;
    });

    let domain = $derived(serviceDomain(dn, origin.domain));

    let logoErrors = $state<string[]>([]);
    let certificateErrors = $state<string[]>([]);

    async function readUpload(
        e: Event,
    ): Promise<{ bytes: Uint8Array<ArrayBuffer>; text: string } | null> {
        const input = e.currentTarget as HTMLInputElement;
        const file = input.files?.[0];
        // Let the same file be picked again after fixing it.
        input.value = "";
        if (!file) return null;

        const bytes = new Uint8Array(await file.arrayBuffer());
        return { bytes, text: new TextDecoder().decode(bytes) };
    }

    async function onLogo(e: Event) {
        const upload = await readUpload(e);
        if (!upload) return;

        logoErrors = logoProblems(upload.text, upload.bytes.length);
        if (logoErrors.length === 0) value.logo = encodeBase64(upload.bytes);
    }

    async function onCertificate(e: Event) {
        const upload = await readUpload(e);
        if (!upload) return;

        certificateErrors = certificateProblems(upload.text, upload.bytes.length);
        if (certificateErrors.length === 0) value.vmc = encodeBase64(upload.bytes);
    }

    function removeCertificate() {
        value.vmc = undefined;
        certificateErrors = [];
    }

    // The value may come from an import or a hand edit: not necessarily base64.
    let certificateCount = $derived.by(() => {
        try {
            return value.vmc ? countCertificates(new TextDecoder().decode(decodeBase64(value.vmc))) : 0;
        } catch {
            return 0;
        }
    });

    // The record the backend will generate. Writing it to the TXT as well
    // keeps the stored service in step with what gets published; when the
    // browser cannot hash (no secure context), the backend still does.
    let preview = $state<string | null>(null);
    $effect(() => {
        let logo: Uint8Array<ArrayBuffer>;
        let certificate: Uint8Array<ArrayBuffer>;
        try {
            logo = decodeBase64(value.logo);
            certificate = decodeBase64(value.vmc);
        } catch {
            preview = null;
            return;
        }
        const forDomain = domain;

        // A newer run supersedes this one, whatever order they settle in.
        let stale = false;
        Promise.all([contentName(logo, ".svg"), contentName(certificate, ".pem")]).then(
            ([logoName, certificateName]) => {
                if (stale) return;
                if (!logoName || certificateName === null) {
                    preview = null;
                    return;
                }
                preview = generatedRecord(forDomain, logoName, certificateName);
                if (value.txt) value.txt.Txt = preview;
            },
        );
        return () => {
            stale = true;
        };
    });
</script>

<div>
    <h4 class="text-primary pb-1 border-bottom border-1">
        {$t("services.bimi-hosted.title", {
            default: "Brand Indicators for Message Identification",
        })}
    </h4>
    <p class="text-muted small mb-1">
        {$t("services.bimi-hosted.intro", {
            default:
                "Displays your brand logo next to your emails in the mail clients that support it. HappyDomain hosts the logo and its certificate, and generates the BIMI record pointing to them.",
        })}
    </p>
    <p class="text-muted small">
        {$t("services.bimi-hosted.requirements", {
            default:
                "Most mail providers (Gmail, Yahoo, Apple Mail) only display the logo with a Verified Mark Certificate or a Common Mark Certificate, and for mail passing a DMARC policy at quarantine or reject.",
        })}
    </p>

    <BasicInput
        edit
        index="selector"
        specs={{
            id: "selector",
            label: $t("services.bimi-hosted.selector", { default: "Selector" }),
            placeholder: "default",
            type: "string",
            required: true,
            description: $t("services.bimi-hosted.selector-desc", {
                default:
                    "Name of the BIMI record. Use 'default' unless you publish multiple logos.",
            }),
        }}
        bind:value={selector}
    />

    <FormGroup row>
        <Label md="4" class="text-md-end text-primary">
            {$t("services.bimi-hosted.logo", { default: "Logo" })}
        </Label>
        <div class="col-md-8">
            {#if value.logo}
                <!-- An <img> never runs what the SVG might carry, unlike an
                     inlined SVG: the file is the user's, not ours. -->
                <img
                    src={"data:image/svg+xml;base64," + value.logo}
                    alt={$t("services.bimi-hosted.logo", { default: "Logo" })}
                    class="border rounded mb-2 d-block"
                    style="width: 96px; height: 96px; object-fit: contain"
                />
            {:else}
                <p class="small text-muted fst-italic mb-2">
                    {$t("services.bimi-hosted.logo-none", { default: "No logo uploaded yet." })}
                </p>
            {/if}
            <label class="btn btn-sm btn-outline-primary">
                <input type="file" accept=".svg,image/svg+xml" onchange={onLogo} hidden />
                {$t("services.bimi-hosted.logo-upload", { default: "Upload a logo (SVG)" })}
            </label>
            {#each logoErrors as error (error)}
                <p class="small text-danger mt-1 mb-0">{$t("services.bimi-hosted." + error)}</p>
            {/each}
            <p class="small text-muted mt-2 mb-0">
                {$t("services.bimi-hosted.logo-desc", {
                    default:
                        'SVG file using the SVG Tiny Portable/Secure profile (baseProfile="tiny-ps"), square, at most 32 KiB.',
                })}
            </p>
        </div>
    </FormGroup>

    <FormGroup row>
        <Label md="4" class="text-md-end text-primary">
            {$t("services.bimi-hosted.certificate", { default: "Mark certificate" })}
        </Label>
        <div class="col-md-8">
            <p
                class="small mb-2"
                class:text-muted={!certificateCount}
                class:fst-italic={!certificateCount}
            >
                {#if certificateCount}
                    {$t("services.bimi-hosted.certificate-loaded", { count: certificateCount })}
                {:else}
                    {$t("services.bimi-hosted.certificate-none", {
                        default: "No certificate: most mail providers will not display the logo.",
                    })}
                {/if}
            </p>
            <div class="d-flex gap-2">
                <label class="btn btn-sm btn-outline-primary mb-0">
                    <input
                        type="file"
                        accept=".pem,.crt,application/pem-certificate-chain"
                        onchange={onCertificate}
                        hidden
                    />
                    {$t("services.bimi-hosted.certificate-upload", {
                        default: "Upload a certificate (PEM)",
                    })}
                </label>
                {#if value.vmc}
                    <Button size="sm" color="danger" outline onclick={removeCertificate}>
                        {$t("services.bimi-hosted.certificate-remove", {
                            default: "Remove the certificate",
                        })}
                    </Button>
                {/if}
            </div>
            {#each certificateErrors as error (error)}
                <p class="small text-danger mt-1 mb-0">{$t("services.bimi-hosted." + error)}</p>
            {/each}
            <p class="small text-muted mt-2 mb-0">
                {$t("services.bimi-hosted.certificate-desc", {
                    default:
                        "Verified Mark Certificate (VMC) or Common Mark Certificate (CMC) issued for your logo, in PEM format, with its chain.",
                })}
            </p>
        </div>
    </FormGroup>

    <h5 class="mt-3 text-primary pb-1 border-bottom border-1">
        {$t("services.bimi-hosted.hosting", { default: "Hosting" })}
    </h5>
    <p class="small text-muted">
        {$t("services.bimi-hosted.hosting-desc", { domain })}
    </p>

    <h5 class="mt-3 text-primary pb-1 border-bottom border-1">
        {$t("services.bimi-hosted.preview", { default: "Generated record" })}
    </h5>
    {#if preview}
        <pre class="p-2 small mb-0 text-wrap text-break"><code
                >{selector}._bimi.{domain}. IN TXT "{preview}"</code
            ></pre>
    {:else}
        <p class="small text-muted fst-italic mb-0">
            {$t("services.bimi-hosted.preview-unavailable", {
                default: "The record is generated when the zone is saved.",
            })}
        </p>
    {/if}
</div>
