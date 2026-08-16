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
    import ImgWithFallback from "$lib/components/ImgWithFallback.svelte";

    interface Props {
        domain: string;
        style?: string;
        [key: string]: unknown;
    }

    let {
        domain,
        style = "width: 2.5em; height: 2.5em; object-fit: scale-down;",
        ...rest
    }: Props = $props();

    // Strip trailing dot from FQDN for favicon lookup
    let cleanDomain = $derived(domain.replace(/\.$/, ""));

    // The www. prefix says nothing about the site, so drop it from the
    // monogram fallback's label; the favicon lookup itself still uses the
    // FQDN as given.
    let label = $derived((cleanDomain || domain).replace(/^www\./i, ""));
</script>

<ImgWithFallback
    src={cleanDomain ? "/api/favicon/" + encodeURIComponent(cleanDomain) : undefined}
    errorKey={cleanDomain}
    {label}
    loading="lazy"
    {style}
    {...rest}
/>
