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
        title?: string;
    }

    let { class: className, color = "primary", size = "", title }: Props = $props();

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
            case "en":
            case "fr":
                // Authenticated dashboard / landing.
                return "/pages/domains/";

            case "login":
            case "forgotten-password":
                return "/pages/login/";

            case "register":
            case "email-validation":
                return "/pages/signup/";

            case "me":
                if (path[2] == "notifications") return "/pages/notifications/";
                return "/pages/me/";

            case "providers":
                if (path[2] == "features") return "/pages/provider-features/";
                if (path[2] == "new") return "/pages/provider-new-choice/";
                // /providers/[prvid] and /providers/[prvid]/domains
                if (path.length > 2) return "/pages/provider-update/";
                return "/pages/provider-list/";

            case "domains":
                // /domains
                if (path.length <= 2) return "/pages/domains/";
                // /domains/new and /domains/new/[dn]
                if (path[2] == "new") return "/pages/domain-new/";
                // /domains/[dn]/...
                switch (path[3]) {
                    case "history":
                    case "logs":
                        return "/pages/domain-history/";
                    case "import_zone":
                        return "/pages/import-export/";
                    case "checks":
                    case "checkers":
                        return "/pages/checks/";
                    case "[[historyid]]":
                        // /domains/[dn]/[[historyid]]/export
                        if (path[4] == "export") return "/pages/import-export/";
                        // /domains/[dn]/[[historyid]]/[subdomain]/...
                        if (path[4] == "[subdomain]") {
                            if (path[5] == "[serviceid]") {
                                if (path[6] == "checks" || path[6] == "checkers")
                                    return "/pages/checks/";
                                return "/pages/services/";
                            }
                            return "/pages/subdomains/";
                        }
                        // Bare zone editor.
                        return "/pages/domain-abstract/";
                    default:
                        return "/pages/domain-abstract/";
                }

            case "availability":
            case "whois":
                return "/pages/domain-availability/";

            case "resolver":
                return "/pages/tools-client/";

            case "checkers":
                return "/pages/checks/";

            case "generator":
                return "/pages/services/";

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
