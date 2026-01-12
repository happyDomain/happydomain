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
    import OrphanEditor from '$lib/components/services/editors/svcs.Orphan.svelte';

    interface Props {
        dn: string;
        origin: Domain;
        type: string;
        value: any;
    }

    let { dn, origin, type, value = $bindable({}) }: Props = $props();

    // Map of all editor modules (lazy loaded)
    const editorModules = import.meta.glob('./editors/*.svelte');

    // Dynamically load the appropriate editor component
    let componentPromise = $derived(
        (async () => {
            const filename = `${type}.svelte`;
            const path = `./editors/${filename}`;

            if (editorModules[path]) {
                const module = await editorModules[path]() as { default: any };
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
    <EditorComponent
        {dn}
        {origin}
        {type}
        bind:value={value}
    />
{:catch error}
    <div class="alert alert-warning">
        <p>Failed to load editor for type: {type}</p>
        <p class="small text-muted">Error: {error.message}</p>
    </div>
    <OrphanEditor {dn} {origin} {type} bind:value={value} />
{/await}
