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
    import { Button, Icon, type Color } from "@sveltestrap/sveltestrap";

    import { t } from "$lib/translations";

    export { className as class };
    let className: string = "";
    export let color: Color | 'link' | string = "primary";
    export let href: string;
    export let size: 'sm' | 'lg' | string = "";
    export let title: string | null = null;
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
