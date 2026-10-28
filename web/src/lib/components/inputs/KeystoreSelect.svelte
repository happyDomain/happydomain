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
    import type { Keystore } from "$lib/api/keystores";
    import { effectiveKeystore, eligibleKeystores, keystores } from "$lib/stores/keystores";
    import { toasts } from "$lib/stores/toasts";
    import { t } from "$lib/translations";

    import { useKeystoreChoice } from "./keystoreChoice.svelte";

    interface Props {
        index: string;
    }

    let { index }: Props = $props();

    const choice = useKeystoreChoice();

    let eligible: Array<Keystore> = $derived(
        $keystores && choice
            ? eligibleKeystores($keystores, choice.unattended).filter(
                  (ks) => ks.ref !== choice.exclude,
              )
            : [],
    );

    // Where the secrets are, or go without any choice: the keystore the object
    // already uses, else the default one.
    // The keystore excluded gives way to the policy, as on the server.
    let current: Keystore | undefined = $derived.by(() => {
        if (!$keystores || !choice) return undefined;
        const ks = effectiveKeystore(
            $keystores,
            choice.objectType,
            choice.objectId,
            "",
            choice.unattended,
        );
        return ks?.ref === choice.exclude ? undefined : ks;
    });

    function label(ks: Keystore): string {
        return ks.scope === "instance"
            ? $t("keystores.instance", { kind: ks.kind ?? "" })
            : (ks.name ?? "");
    }

    function select(e: Event) {
        if (!choice || !$keystores) return;
        const ref = (e.target as HTMLSelectElement).value;
        const before = effectiveKeystore(
            $keystores,
            choice.objectType,
            choice.objectId,
            choice.value,
            choice.unattended,
        );
        // Back to where the secrets already are: nothing to move.
        choice.value = ref === current?.ref ? "" : ref;
        const after = $keystores.find((ks) => ks.ref === ref);

        // A reference is no secret, nor the other way round: whatever the
        // fields hold has to be entered again.
        if (Boolean(before?.references) !== Boolean(after?.references)) {
            choice.resetSecrets++;
            toasts.addToast({
                title: $t("keystores.reentry-title"),
                message: after?.references
                    ? $t("keystores.reentry-references")
                    : $t("keystores.reentry-secrets"),
                type: "info",
                timeout: 5000,
            });
        }
    }
</script>

{#if choice && eligible.length > 1}
    <select
        id={"keystore-" + index}
        class="form-select form-select-sm flex-grow-0 w-auto"
        title={$t("keystores.choose")}
        aria-label={$t("keystores.choose")}
        value={choice.value || current?.ref || ""}
        onchange={select}
    >
        {#if !current}
            <option value="">{$t("keystores.policy")}</option>
        {/if}
        {#each eligible as ks (ks.ref)}
            <option value={ks.ref}>&#x1F512; {label(ks)}</option>
        {/each}
    </select>
{/if}
