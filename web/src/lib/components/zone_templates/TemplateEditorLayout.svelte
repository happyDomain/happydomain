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
    import { onMount } from "svelte";
    import { goto } from "$app/navigation";
    import { templateLinks } from "$links";

    import { Col, Container, Row, Spinner } from "@sveltestrap/sveltestrap";

    import AliasModal from "$lib/components/modals/Alias.svelte";
    import RecordModal from "$lib/components/modals/Record.svelte";
    import NewServicePath from "$lib/components/services/NewServicePath.svelte";
    import { provideZoneEditor } from "$lib/components/zones/editor";
    import NewSubdomainPath from "$lib/components/zones/NewSubdomainPath.svelte";
    import ServiceDetailsOffcanvas from "$lib/components/zones/ServiceDetailsOffcanvas.svelte";
    import { templateAsZone, templateOrigin, type ZoneTemplate } from "$lib/model/zone_template";
    import { thisZone } from "$lib/stores/thiszone";
    import { thisTemplate, upsertZoneTemplate, zoneTemplates } from "$lib/stores/zone_templates";
    import { t } from "$lib/translations";
    import type { ZoneTemplateBackend } from "./backend";
    import { templateZoneEditor } from "./editor";
    import TemplateSidebar from "./TemplateSidebar.svelte";

    interface Props {
        // The template as loaded by the page.
        template: ZoneTemplate;
        backend: ZoneTemplateBackend;
        // Nothing can be changed, as a global template shown to a user.
        readOnly: boolean;
        // The service being edited, when on its page.
        service?: { subdomain: string; serviceid: string };
        children?: Snippet;
    }

    let { template, backend, readOnly, service, children }: Props = $props();

    function saved(t: ZoneTemplate) {
        thisTemplate.set(t);
        upsertZoneTemplate(t);
    }

    provideZoneEditor(
        templateZoneEditor(
            () => backend,
            () => readOnly,
            saved,
        ),
    );

    // The zone components read thisZone: feed it the template.
    $effect(() => {
        thisTemplate.set(template);
        thisZone.set(templateAsZone(template));
    });

    // The templates the sidebar switches between.
    onMount(() => {
        backend.list().then(
            (list) => zoneTemplates.set(list),
            () => {
                // The sidebar then only offers the template shown.
            },
        );
    });

    let current = $derived($thisTemplate?.id === template.id ? $thisTemplate : template);
    const origin = templateOrigin();

    function select(id: string) {
        if (id !== template.id) goto(templateLinks().template(encodeURIComponent(id)));
    }
</script>

<Container fluid class="d-flex flex-column flex-fill">
    <Row class="flex-fill">
        <Col
            sm={4}
            md={3}
            class="py-2 sticky-top d-flex flex-column"
            style="background-color: #edf5f2; z-index: 0; max-height: 100vh; overflow-y: auto;"
        >
            <TemplateSidebar template={current} {readOnly} {service} onselect={select} />
        </Col>

        <div class="col-sm-8 col-md-9 d-flex">
            {#if $thisZone && $thisZone.id === template.id}
                {@render children?.()}
            {:else}
                <div class="mt-4 text-center flex-fill">
                    <Spinner />
                    <p>{$t("wait.loading")}</p>
                </div>
            {/if}
        </div>
    </Row>
</Container>

{#if $thisZone && $thisZone.id === template.id}
    <NewSubdomainPath {origin} />
    <NewServicePath {origin} zone={$thisZone} historyId={template.id} />
    <RecordModal {origin} zone={$thisZone} />
    <AliasModal {origin} />
    <ServiceDetailsOffcanvas domain={origin} selectedHistory={template.id} />
{/if}
