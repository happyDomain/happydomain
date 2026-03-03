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
    import { page } from "$app/state";
    import { Button, Icon, type Color } from "@sveltestrap/sveltestrap";

    import { helpLinkOverride } from "$lib/stores/help";
    import { providersSpecs } from "$lib/stores/providers";
    import { locale, t } from "$lib/translations";

    interface Props {
        class?: string;
        color?: Color | "link" | string;
        size?: "sm" | "lg" | string;
        title?: string | null;
    }

    let { class: className = "", color = "primary", size = "", title = null }: Props = $props();

    function getHelpPathFromProvider(ptype: string): string {
        if ($providersSpecs && $providersSpecs[ptype]) {
            return $providersSpecs[ptype].helplink;
        } else {
            return "https://help.happydomain.org/";
        }
    }

    function getHelpPathFromRoute(routeId: string): string {
        const path = routeId.split("/");

        if (path.length < 2) return "/";

        switch (path[1]) {
            case "":
                return "/pages/home/";
            case "providers":
                if (path.length > 2) {
                    if (path[2] == "new") return "/pages/source-new-choice/";
                    return "/pages/source-update/";
                }
                return "/pages/source-list/";
            case "domains":
                if (path.length == 2) return "/pages/home/";
                if (path.length > 3 && path[3] == "new") return "/pages/domain-new/";
                return "/pages/domain-abstract/";
            case "me":
                return "/pages/me/";
            case "resolver":
                return "/pages/tools-client/";
            default:
                return "/";
        }
    }

    let href = $derived(
        $helpLinkOverride !== null
            ? "https://help.happydomain.org/" +
                  encodeURIComponent($locale) +
                  "/" +
                  $helpLinkOverride
            : page.route && page.route.id
              ? page.route.id.startsWith("/providers/new/[ptype]")
                  ? getHelpPathFromProvider(page.url.pathname.split("/")[3])
                  : "https://help.happydomain.org/" +
                    encodeURIComponent($locale) +
                    getHelpPathFromRoute(page.route.id)
              : "https://help.happydomain.org/" + encodeURIComponent($locale),
    );
</script>

<Button
    {href}
    target="_blank"
    {color}
    {size}
    class={className}
    {title}
    data-umami-event="help"
    data-umami-event-href={href.substring(href.lastIndexOf("/") - 2)}
>
    <Icon name="question-circle-fill" title={$t("common.help")} />
</Button>
