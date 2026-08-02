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
    import type { Domain } from "$lib/model/domain";
    import OrphanEditor from '$lib/services/svcs.Orphan/editor.svelte';
    import EditorCompliance from '$lib/components/services/EditorCompliance.svelte';

    interface Props {
        dn: string;
        origin: Domain;
        type: string;
        value: Record<string, any>;
    }

    let { dn, origin, type, value = $bindable({}) }: Props = $props();

    // Map of all editor modules (lazy loaded). Each service owns a folder named
    // after its service type, so the type resolves to a path directly.
    const editorModules = import.meta.glob('$lib/services/*/editor.svelte');

    // Index the editors by the name of the folder holding them, which is the
    // service type, whatever shape the bundler gives to the glob keys.
    const editors: Record<string, () => Promise<unknown>> = {};
    for (const [path, load] of Object.entries(editorModules)) {
        const svctype = path.split("/").at(-2);
        if (svctype) editors[svctype] = load;
    }

    // Dynamically load the appropriate editor component
    let componentPromise = $derived(
        (async () => {
            if (editors[type]) {
                const module = await editors[type]() as { default: typeof OrphanEditor };
                return module.default;
            }

            // Fallback to Orphan editor for unknown types
            return OrphanEditor;
        })()
    );
</script>

{#await componentPromise}
    <div class="text-center p-3">
        <div class="spinner-border spinner-border-sm text-primary" role="status">
            <span class="visually-hidden">Loading editor...</span>
        </div>
    </div>
{:then EditorComponent}
    {#key value}
        <EditorComponent
            {dn}
            {origin}
            {type}
            bind:value={value}
        />
    {/key}
    <EditorCompliance {dn} {origin} {type} {value} />
{:catch error}
    <div class="alert alert-warning">
        <p>Failed to load editor for type: {type}</p>
        <p class="small text-muted">Error: {error.message}</p>
    </div>
    <OrphanEditor {dn} {origin} {type} bind:value={value} />
{/await}
