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
    interface Props {
        domain: string;
        style?: string;
        [key: string]: unknown;
    }

    let {
        domain,
        style = "max-width: 100%; max-height: 2.5em",
        ...rest
    }: Props = $props();

    let error = $state(false);

    // Strip trailing dot from FQDN for favicon lookup
    let cleanDomain = $derived(domain.replace(/\.$/, ""));
</script>

{#if !error && cleanDomain}
    <img
        src={"/api/favicon/" + encodeURIComponent(cleanDomain)}
        alt={cleanDomain}
        title={cleanDomain}
        {style}
        {...rest}
        onerror={() => (error = true)}
    />
{:else}
    <span {style} {...rest}></span>
{/if}
