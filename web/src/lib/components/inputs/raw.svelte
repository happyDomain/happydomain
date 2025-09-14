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
    import { createEventDispatcher } from "svelte";

    import { Input, InputGroup, InputGroupText, type InputType } from "@sveltestrap/sveltestrap";

    import type { Field } from "$lib/model/custom_form.svelte";
    import { t } from "$lib/translations";

    const dispatch = createEventDispatcher();

    interface Props {
        edit?: boolean;
        index: string;
        specs?: Field;
        value: any;
        [key: string]: any
    }

    let {
        edit = false,
        index,
        specs = { id: "", type: "" },
        value = $bindable(),
        ...rest
    }: Props = $props();
    let val: any = $state(value);

    let unit: string | null = $derived(specs.type === "time.Duration" || specs.type === "common.Duration" ? "s" : null);

    let inputtype: InputType = $derived((specs.type && (specs.type.startsWith("uint") || specs.type.startsWith("int"))) ? "number" : (specs.type && specs.type === "bool" ? "checkbox" : (specs.textarea ? "textarea" : "text")));

    let inputmax: number | undefined = $derived(computeInputmax(specs));
    let inputmin: number | undefined = $derived(computeInputmin(specs));

    function computeInputmax(specs: any) {
        if (specs.type) {
            if (specs.type == "int8" || specs.type == "uint8") return 255;
            else if (specs.type == "int16" || specs.type == "uint16") return 65536;
            else if (
                specs.type == "int" ||
                specs.type == "uint" ||
                specs.type == "int32" ||
                specs.type == "uint32"
            )
                return 2147483647;
            else if (
                specs.type == "time.Duration" ||
                specs.type == "common.Duration" ||
                specs.type == "int64" ||
                specs.type == "uint64"
            )
                return 9007199254740991;
        }
        return undefined;
    }
    function computeInputmin(specs: any) {
        if (inputmax) {
            if (specs.type && specs.type.startsWith("uint")) return 0;
            else return -inputmax - 1;
        }
        return undefined;
    }

    function checkBase64(val: string): boolean {
        try {
            atob(val);
            return true;
        } catch {
            return false;
        }
    }

    let feedback: string | undefined = $derived(computeFeedback(value));
    function computeFeedback(val: any) {
        if (inputmax && val > inputmax) {
            return t.get("errors.too-high", { max: inputmax });
        } else if (inputmin && val < inputmin) {
            return t.get("errors.too-low", { min: inputmin });
        } else if (
            specs.type &&
            (specs.type === "[]uint8" || specs.type === "[]byte") &&
            !checkBase64(val)
        ) {
            if (checkBase64(val + "==")) {
                return
                    t.get("errors.base64") +
                    " " +
                    t.get("errors.suggestion", { suggestion: `${val}==` });
            } else if (checkBase64(val + "=")) {
                return
                    t.get("errors.base64") +
                    " " +
                    t.get("errors.suggestion", { suggestion: `${val}=` });
            } else if (checkBase64(val + "a")) {
                return t.get("errors.base64") + " " + t.get("errors.base64-unfinished");
            } else {
                return t.get("errors.base64") + " " + t.get("errors.base64-illegal-char");
            }
        }
        return undefined;
    }

    function parseValue(e: InputEvent) {
        if (e.target && 'value' in e.target) {
            const target = e.target as HTMLInputElement;
            val = target.value;

            if (
                specs.type &&
                (specs.type.startsWith("int") ||
                 specs.type.startsWith("uint") ||
                 specs.type == "time.Duration" ||
                 specs.type == "common.Duration")
            ) {
                if (target.value.length != 0 && target.value == parseInt(target.value, 10).toString()) {
                    value = parseInt(target.value, 10);
                } else if (specs.type == "time.Duration" || specs.type == "common.Duration") {
                    // Allow 1m, 1s, ...
                    value = val;
                }
            } else {
                value = val;
            }
        }
    }
</script>

<InputGroup size="sm" {...rest}>
    {#if edit && specs.choices && specs.choices.length > 0}
        <Input
            id={"spec-" + index + "-" + specs.id}
            type="select"
            required={specs.required}
            bind:value
            on:focus={() => dispatch("focus")}
            on:blur={() => dispatch("blur")}
        >
            {#each specs.choices as opt}
                <option value={opt}>{opt}</option>
            {/each}
        </Input>
    {:else if inputtype === "checkbox"}
        <Input
            id={"spec-" + index + "-" + specs.id}
            type={inputtype}
            class="fw-bold"
            {feedback}
            invalid={feedback !== undefined}
            placeholder={specs.placeholder}
            plaintext={!edit}
            readonly={!edit}
            required={specs.required}
            bind:checked={value}
            on:focus={() => dispatch("focus")}
            on:blur={() => dispatch("blur")}
        />
    {:else}
        <Input
            id={"spec-" + index + "-" + specs.id}
            type={inputtype}
            class="fw-bold"
            {feedback}
            invalid={feedback !== undefined}
            min={inputmin}
            max={inputmax}
            placeholder={specs.placeholder}
            plaintext={!edit}
            readonly={!edit}
            required={specs.required}
            bind:value={val}
            on:focus={() => dispatch("focus")}
            on:blur={() => dispatch("blur")}
            on:input={parseValue}
        />
    {/if}

    {#if unit !== null}
        <InputGroupText>{unit}</InputGroupText>
    {/if}
</InputGroup>
