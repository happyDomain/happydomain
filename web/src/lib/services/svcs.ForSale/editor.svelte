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

    import type { Domain } from "$lib/model/domain";
    import { t } from "$lib/translations";
    import {
        byteLength,
        FORSALE_MAX_VALUE_LEN,
        FORSALE_TAGS,
        ForSaleService,
        type ForSaleTag,
        type ForSaleValue,
        newForSaleRecord,
        parsePrice,
        stringifyPrice,
    } from "./model.svelte";

    interface Props {
        dn: string;
        origin: Domain;
        readonly?: boolean;
        value: ForSaleValue;
    }

    let { readonly = false, value = $bindable({}) }: Props = $props();

    // The RRset is the source of truth; normalize it once so the class below can
    // splice it in place.
    if (!value.txt) {
        value.txt = [newForSaleRecord(null, "")];
    } else if (!Array.isArray(value.txt)) {
        value.txt = [value.txt];
    } else if (value.txt.length === 0) {
        value.txt.push(newForSaleRecord(null, ""));
    }

    let val = $derived(new ForSaleService(value));

    // A few widespread currencies, offered as suggestions only: RFC 10023 sec. 3.3
    // recommends ISO 4217 but does not restrict the value to it.
    const CURRENCIES = ["EUR", "USD", "GBP", "CHF", "CAD", "AUD", "JPY", "CNY", "XBT"];

    const ICONS: Record<ForSaleTag, string> = {
        fval: "tag",
        ftxt: "chat-left-text",
        furi: "link-45deg",
        fcod: "upc-scan",
    };

    function updatePrice(index: number, part: "currency" | "amount", raw: string): void {
        const price = parsePrice(val.getValue(index));
        if (part === "currency") {
            price.currency = raw.toUpperCase().replace(/[^A-Z]/g, "");
        } else {
            price.amount = raw;
        }
        val.setValue(index, stringifyPrice(price.currency, price.amount));
    }
</script>

<p class="text-muted small mb-3">
    {$t("resources.FORSALE.intro")}
</p>

{#each val.editableEntries as entry (entry.index)}
    {@const tag = entry.pair.tag}
    {@const overflow =
        (tag === "ftxt" || tag === "fcod") && byteLength(entry.pair.value) > FORSALE_MAX_VALUE_LEN}
    <div class="mb-2">
        <InputGroup size="sm">
            <InputGroupText class="forsale-kind">
                {#if tag && tag in ICONS}
                    <Icon name={ICONS[tag as ForSaleTag]} class="me-1" />
                    {$t(`resources.FORSALE.${tag}`)}
                {:else}
                    <Icon name="question-circle" class="me-1" />
                    {tag ?? "?"}
                {/if}
            </InputGroupText>

            {#if tag === "fval"}
                {@const price = parsePrice(entry.pair.value)}
                <Input
                    style="max-width: 6rem"
                    list="forsale-currencies"
                    {readonly}
                    placeholder={$t("resources.FORSALE.price-currency")}
                    value={price.currency}
                    oninput={(e: Event) =>
                        updatePrice(entry.index, "currency", (e.target as HTMLInputElement).value)}
                />
                <Input
                    {readonly}
                    inputmode="decimal"
                    placeholder={$t("resources.FORSALE.price-amount")}
                    value={price.amount}
                    oninput={(e: Event) =>
                        updatePrice(entry.index, "amount", (e.target as HTMLInputElement).value)}
                />
            {:else if entry.pair.malformed}
                <Input
                    {readonly}
                    invalid
                    value={entry.pair.value}
                    oninput={(e: Event) =>
                        val.setRaw(entry.index, (e.target as HTMLInputElement).value)}
                />
            {:else}
                <Input
                    {readonly}
                    invalid={overflow}
                    type={tag === "furi" ? "url" : "text"}
                    placeholder={tag === "furi"
                        ? $t("resources.FORSALE.uri-placeholder")
                        : tag === "fcod"
                          ? $t("resources.FORSALE.code-placeholder")
                          : $t("resources.FORSALE.text-placeholder")}
                    value={entry.pair.value}
                    oninput={(e: Event) =>
                        val.setValue(entry.index, (e.target as HTMLInputElement).value)}
                />
            {/if}

            {#if !readonly}
                <Button
                    type="button"
                    color="danger"
                    outline
                    title={$t("common.delete")}
                    onclick={() => val.remove(entry.index)}
                >
                    <Icon name="trash" />
                </Button>
            {/if}
        </InputGroup>

        {#if overflow}
            <small class="text-danger">
                {$t("resources.FORSALE.too-long", {
                    count: byteLength(entry.pair.value),
                    max: FORSALE_MAX_VALUE_LEN,
                })}
            </small>
        {/if}
    </div>
{/each}

<datalist id="forsale-currencies">
    {#each CURRENCIES as currency (currency)}
        <option value={currency}></option>
    {/each}
</datalist>

{#if !readonly}
    <div class="d-flex flex-wrap align-items-center gap-2 mt-3">
        <span class="text-muted small">{$t("resources.FORSALE.add")}</span>
        {#each FORSALE_TAGS as tag (tag)}
            <Button
                type="button"
                color="primary"
                outline
                size="sm"
                onclick={() => val.add(tag, tag === "fval" ? "EUR" : "")}
            >
                <Icon name={ICONS[tag]} />
                {$t(`resources.FORSALE.${tag}`)}
            </Button>
        {/each}
    </div>
{/if}

<style>
    :global(.forsale-kind) {
        min-width: 7.5rem;
    }
</style>
