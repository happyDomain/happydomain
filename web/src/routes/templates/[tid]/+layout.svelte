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
    import type { Snippet } from "svelte";
    import { page } from "$app/state";

    import { userTemplateBackend } from "$lib/components/zone_templates/backend";
    import TemplateEditorLayout from "$lib/components/zone_templates/TemplateEditorLayout.svelte";
    import { isGlobalTemplate, type ZoneTemplate } from "$lib/model/zone_template";

    interface Props {
        data: { template: ZoneTemplate };
        children?: Snippet;
    }

    let { data, children }: Props = $props();

    // The service being edited, on its page.
    let service = $derived(
        page.params.serviceid
            ? {
                  subdomain: page.params.subdomain === "@" ? "" : (page.params.subdomain ?? ""),
                  serviceid: page.params.serviceid,
              }
            : undefined,
    );
</script>

<svelte:head>
    <title>{data.template.name} - HappyDomain</title>
</svelte:head>

<TemplateEditorLayout
    template={data.template}
    backend={userTemplateBackend}
    readOnly={isGlobalTemplate(data.template)}
    {service}
>
    {@render children?.()}
</TemplateEditorLayout>
