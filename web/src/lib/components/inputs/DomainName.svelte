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
    import { Button, Icon, Input, InputGroup, InputGroupText } from "@sveltestrap/sveltestrap";

    import { domainNameIssue, domainNameIssueBlocks, fqdn } from "$lib/dns";
    import { t } from "$lib/translations";
    import { looksAbsolute, textToDomainName } from "$lib/utils/domainname";

    import DomainNameIssue from "./DomainNameIssue.svelte";

    interface Props {
        autofocus?: boolean;
        id?: string;
        // What the name is relative to: the zone for record data, the
        // subdomain of the service for owner names.
        origin: string;
        placeholder?: string;
        readonly?: boolean;
        required?: boolean;
        size?: "sm" | "md" | "lg";
        // Whether the name is acceptable, for forms to hold their submission.
        valid?: boolean;
        // Name as stored: relative to origin unless it ends with a dot.
        value: string | undefined;
        // Only names within origin are accepted, as for owner names.
        within?: boolean;
        onfocus?: () => void;
        onblur?: () => void;
    }

    let {
        autofocus = false,
        id,
        origin,
        placeholder,
        readonly = false,
        required = false,
        size = "sm",
        // Only written for the parent, which the rule cannot see.
        // eslint-disable-next-line no-useless-assignment
        valid = $bindable(),
        value = $bindable(),
        within = false,
        onfocus,
        onblur,
    }: Props = $props();

    let zone = $derived(origin.replace(/\.$/, ""));

    let text = $state(value ?? "");
    let emitted: unknown = value;

    // The value changed from elsewhere (another field, a reset): show it.
    $effect(() => {
        if (value !== emitted) {
            emitted = value;
            text = value ?? "";
        }
    });

    let name: string = $derived(textToDomainName(text));
    let absolute: string = $derived(name === "" ? "" : fqdn(name, zone ? zone + "." : ""));
    // The root stands for "none" where a name is expected: a null MX (RFC
    // 7505), a NAPTR without replacement.
    let root = $derived(name === ".");
    let issue = $derived(root ? null : domainNameIssue(absolute));
    let blocks = $derived(domainNameIssueBlocks(issue));

    // Whether an absolute name is the zone or lies under it.
    function inZone(abs: string): boolean {
        const n = abs.toLowerCase();
        const z = zone.toLowerCase() + ".";
        return n === z || n.endsWith("." + z);
    }

    let relative = $derived(zone !== "" && !name.endsWith("."));
    let outside = $derived(zone !== "" && !relative && !root && !inZone(name));
    let refused = $derived(blocks || (within && (outside || root)));
    // Without its final dot, a name is relative to the zone, as in a zone
    // file: www.example.com is then www.example.com.example.com.
    let suggestion: string | null = $derived(
        relative &&
            (looksAbsolute(name) || inZone(name + ".")) &&
            (!within || inZone(name + "."))
            ? name + "."
            : null,
    );

    $effect(() => {
        valid = name !== "" && !refused;
    });

    function oninput() {
        if (!refused) {
            emitted = name;
            value = name;
        }
    }

    function normalize() {
        if (!refused) text = name;
        onblur?.();
    }

    function useSuggestion() {
        if (suggestion === null) return;
        text = suggestion;
        oninput();
    }
</script>

<div class="flex-fill">
    <InputGroup size={size === "md" ? undefined : size}>
        <Input
            {autofocus}
            {id}
            type="text"
            class="font-monospace"
            invalid={refused}
            {placeholder}
            plaintext={readonly}
            {readonly}
            {required}
            autocomplete="off"
            autocapitalize="off"
            spellcheck="false"
            bind:value={text}
            on:input={oninput}
            on:focus={() => onfocus?.()}
            on:blur={normalize}
        />
        {#if relative && name !== "@"}
            <InputGroupText class="font-monospace">.{zone}.</InputGroupText>
        {/if}
    </InputGroup>
    {#if issue}
        <DomainNameIssue {issue} />
    {:else if within && (outside || root)}
        <div class="form-text text-danger">
            <Icon name="exclamation-circle" />
            {$t("domains.input.must-be-within", { zone })}
        </div>
    {:else if root}
        <div class="form-text">= {$t("domains.input.root")}</div>
    {:else if relative && name !== ""}
        <div class="form-text">
            = <span class="font-monospace">{absolute}</span>
        </div>
    {:else if outside}
        <div class="form-text">{$t("domains.input.outside-zone")}</div>
    {/if}
    {#if suggestion !== null && !readonly}
        <div class="form-text text-warning-emphasis">
            <Icon name="exclamation-triangle" />
            {$t("domains.input.did-you-mean", { suggestion })}
            <Button
                type="button"
                color="link"
                size="sm"
                class="p-0 align-baseline"
                onclick={useSuggestion}
            >
                {$t("domains.input.use")}
            </Button>
        </div>
    {/if}
</div>
