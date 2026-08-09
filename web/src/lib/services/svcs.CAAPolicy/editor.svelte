<!--
     This file is part of the happyDomain (R) project.
     Copyright (c) 2022-2024 happyDomain
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
    import { Button, Icon, Input, type InputType } from "@sveltestrap/sveltestrap";

    import CAAIssuer from "./issuer.svelte";
    import CAAIodef from "./iodef.svelte";
    import type { Domain } from "$lib/model/domain";
    import type { SvcsCAAPolicyBody } from "$lib/services_bodies";
    import { t } from "$lib/translations";
    import { rev_issuers } from "./issuers";
    import {
        CAA_ISSUE_TAGS,
        CAAPolicy,
        type CAAIssueTag,
        type CAAMode,
        type CAATag,
    } from "./model.svelte";

    interface Props {
        dn: string;
        origin: Domain;
        readonly?: boolean;
        value: SvcsCAAPolicyBody;
    }

    let { dn, readonly = false, value = $bindable() }: Props = $props();

    // The RRset is the source of truth; normalize it once so the class below can
    // splice it in place.
    if (!value.caa) {
        value.caa = [];
    } else if (!Array.isArray(value.caa)) {
        value.caa = [value.caa];
    }

    let val = $derived(new CAAPolicy(value, dn));

    const MODES: Array<CAAMode> = ["any", "restricted", "none"];

    const MODE_ICONS: Record<CAAMode, string> = {
        any: "globe",
        restricted: "shield-check",
        none: "shield-slash",
    };

    // A kind of certificate the user just restricted holds no authority yet, so
    // the records still read as "any": remember the choice until one is added.
    const restricting: Partial<Record<CAAIssueTag, boolean>> = $state({});

    function modeOf(tag: CAAIssueTag): CAAMode {
        const mode = val.mode(tag);
        return mode === "any" && restricting[tag] ? "restricted" : mode;
    }

    function setMode(tag: CAAIssueTag, mode: CAAMode): void {
        restricting[tag] = mode === "restricted";
        val.setMode(tag, mode);
    }

    // Wildcard certificates fall back on the regular issuance rule when nothing
    // is published for them, which is not the same as allowing everyone.
    function modeLabel(tag: CAAIssueTag, mode: CAAMode): string {
        if (tag === "issuewild" && mode === "any") return $t("resources.CAA.mode.inherit");
        return $t(`resources.CAA.mode.${mode}`);
    }

    function explain(tag: CAAIssueTag, mode: CAAMode): string {
        if (tag === "issuewild" && mode === "any") return $t("resources.CAA.explain.inherit");
        return $t(`resources.CAA.explain.${mode}`, { kind: $t(`resources.CAA.kinds.${tag}.kind`) });
    }

    /** The name the certificate authority is known under, when we know it. */
    function issuerName(record: string): string {
        const domain = record.split(";")[0].trim();
        if (!domain) return $t("resources.CAA.unnamed-issuer");
        return rev_issuers[domain] ?? domain;
    }

    /** One line telling what the policy does today for that kind of certificate. */
    function summarize(tag: CAAIssueTag, mode: CAAMode): string {
        if (mode !== "restricted") return modeLabel(tag, mode);
        const issuers = val.issuers(tag);
        if (!issuers.length) return $t("resources.CAA.mode.restricted");
        return issuers.map((e) => issuerName(e.record.Value)).join(", ");
    }

    const CONTACTS: Array<{ tag: CAATag; type: InputType; placeholder: string }> = [
        { tag: "contactemail", type: "email", placeholder: "contact@example.com" },
        { tag: "contactphone", type: "tel", placeholder: "+1-555-0123" },
    ];
</script>

<p class="mb-4">
    {$t("resources.CAA.intro")}
</p>

<div class="card mb-5">
    <div class="card-body">
        <h5 class="card-title mb-3">{$t("resources.CAA.summary")}</h5>
        <ul class="list-unstyled mb-0">
            {#each CAA_ISSUE_TAGS as tag (tag)}
                {@const mode = modeOf(tag)}
                <li class="d-flex flex-wrap column-gap-3 py-1">
                    <span class="text-muted caa-summary-kind">
                        {$t(`resources.CAA.kinds.${tag}.title`)}
                    </span>
                    <span class:text-success={mode !== "any"}>
                        <Icon name={MODE_ICONS[mode]} />
                        {summarize(tag, mode)}
                    </span>
                </li>
            {/each}
        </ul>
    </div>
</div>

{#each CAA_ISSUE_TAGS as tag (tag)}
    {@const mode = modeOf(tag)}
    <section class="mb-5">
        <h4 class="mb-2">{$t(`resources.CAA.kinds.${tag}.title`)}</h4>
        <p class="text-muted mb-3">{$t(`resources.CAA.kinds.${tag}.help`)}</p>

        <div class="btn-group" role="group">
            {#each MODES as m (m)}
                <input
                    type="radio"
                    class="btn-check"
                    name="caa-{tag}"
                    id="caa-{tag}-{m}"
                    checked={mode === m}
                    disabled={readonly}
                    onchange={() => setMode(tag, m)}
                />
                <label class="btn btn-outline-primary" for="caa-{tag}-{m}">
                    <Icon name={MODE_ICONS[m]} class="me-1" />
                    {modeLabel(tag, m)}
                </label>
            {/each}
        </div>

        <p class="mt-3 mb-0">{explain(tag, mode)}</p>

        {#if mode === "restricted"}
            <ul class="list-unstyled ms-3 mt-3 mb-0">
                {#each val.issuers(tag) as entry (entry.index)}
                    <li class="mb-2">
                        <CAAIssuer
                            {readonly}
                            bind:flag={val.records[entry.index].Flag}
                            bind:tag={val.records[entry.index].Tag}
                            bind:value={val.records[entry.index].Value}
                            on:delete-issuer={() => {
                                // Removing the last one empties the RRset for
                                // that tag: keep the section open, so the user
                                // sees they are back to allowing everyone.
                                restricting[tag] = true;
                                val.remove(entry.index);
                            }}
                        />
                    </li>
                {/each}
                {#if !readonly}
                    <li>
                        <CAAIssuer newone on:add-issuer={(e) => val.add(tag, e.detail)} />
                    </li>
                {/if}
            </ul>

            {#if !val.issuers(tag).length}
                <p class="text-warning-emphasis mt-3 mb-0">{$t("resources.CAA.pick-one")}</p>
            {/if}
        {/if}
    </section>
{/each}

<section class="mb-5">
    <h4 class="mb-2">{$t("resources.CAA.incident-response")}</h4>
    <p class="text-muted mb-3">{$t("resources.CAA.incident-response-text")}</p>

    {#each val.entries("iodef") as entry (entry.index)}
        <CAAIodef
            {readonly}
            bind:flag={val.records[entry.index].Flag}
            bind:tag={val.records[entry.index].Tag}
            bind:value={val.records[entry.index].Value}
            on:delete-iodef={() => val.remove(entry.index)}
        />
    {/each}
    {#if !readonly}
        <CAAIodef newone on:add-iodef={(e) => val.add("iodef", e.detail)} />
    {/if}
</section>

<section>
    <h4 class="mb-2">{$t("resources.CAA.contact-info")}</h4>
    <p class="text-muted mb-3">{$t("resources.CAA.contact-info-text")}</p>

    {#each CONTACTS as contact (contact.tag)}
        <h5 class="mb-2">{$t(`resources.CAA.${contact.tag}`)}</h5>

        {#each val.entries(contact.tag) as entry (entry.index)}
            <div class="d-flex gap-2 mb-2">
                <Input
                    type={contact.type}
                    {readonly}
                    placeholder={contact.placeholder}
                    bind:value={val.records[entry.index].Value}
                />
                {#if !readonly}
                    <Button
                        type="button"
                        color="danger"
                        outline
                        title={$t("common.delete")}
                        on:click={() => val.remove(entry.index)}
                    >
                        <Icon name="trash" />
                    </Button>
                {/if}
            </div>
        {/each}

        {#if !readonly}
            <Button
                type="button"
                color="primary"
                outline
                class="mb-4"
                on:click={() => val.add(contact.tag, "")}
            >
                <Icon name="plus" class="me-1" />
                {$t(`resources.CAA.add-${contact.tag}`)}
            </Button>
        {/if}
    {/each}
</section>

<style>
    .caa-summary-kind {
        min-width: 18rem;
    }
</style>
