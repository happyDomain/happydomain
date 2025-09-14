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
    import ACMEChallengeEditor from '$lib/components/services/editors/abstract.ACMEChallenge.svelte'
    import DelegationEditor from '$lib/components/services/editors/abstract.Delegation.svelte'
    import GithubOrgVerifEditor from '$lib/components/services/editors/abstract.GithubOrgVerif.svelte'
    import GitlabPageVerifEditor from '$lib/components/services/editors/abstract.GitlabPageVerif.svelte'
    import GoogleVerifEditor from '$lib/components/services/editors/abstract.GoogleVerif.svelte'
    import KeybaseVerifEditor from '$lib/components/services/editors/abstract.KeybaseVerif.svelte'
    import MatrixIMEditor from '$lib/components/services/editors/abstract.MatrixIM.svelte'
    import NsOnlyOriginEditor from '$lib/components/services/editors/abstract.NSOnlyOrigin.svelte'
    import OpenPGPEditor from '$lib/components/services/editors/abstract.OpenPGP.svelte'
    import OriginEditor from '$lib/components/services/editors/abstract.Origin.svelte'
    import RFC6186Editor from '$lib/components/services/editors/abstract.RFC6186.svelte'
    import ScalewayChallengeEditor from '$lib/components/services/editors/abstract.ScalewayChallenge.svelte'
    import ServerEditor from '$lib/components/services/editors/abstract.Server.svelte'
    import SMimeCertEditor from '$lib/components/services/editors/abstract.SMimeCert.svelte'
    import XMPPEditor from '$lib/components/services/editors/abstract.XMPP.svelte'
    import CAAEditor from '$lib/components/services/editors/svcs.CAAPolicy.svelte'
    import CNAMEEditor from '$lib/components/services/editors/svcs.CNAME.svelte'
    import DKIMEditor from '$lib/components/services/editors/svcs.DKIMRecord.svelte'
    import DMARCEditor from '$lib/components/services/editors/svcs.DMARC.svelte'
    import MTA_STSEditor from '$lib/components/services/editors/svcs.MTA_STS.svelte'
    import MXsEditor from '$lib/components/services/editors/svcs.MXs.svelte'
    import NAPTREditor from '$lib/components/services/editors/svcs.NAPTR.svelte'
    import OrphanEditor from '$lib/components/services/editors/svcs.Orphan.svelte'
    import PTREditor from '$lib/components/services/editors/svcs.PTR.svelte'
    import SPFEditor from '$lib/components/services/editors/svcs.SPF.svelte'
    import SSHFPsEditor from '$lib/components/services/editors/svcs.SSHFPs.svelte'
    import TLSAsEditor from '$lib/components/services/editors/svcs.TLSAs.svelte'
    import TLS_RPTEditor from '$lib/components/services/editors/svcs.TLS_RPT.svelte'
    import UnknownSRVEditor from '$lib/components/services/editors/svcs.UnknownSRV.svelte'
    import type { Domain } from "$lib/model/domain";

    interface Props {
        dn: string;
        origin: Domain;
        type: string;
        value: any;
    }

    let { dn, origin, type, value = $bindable({}) }: Props = $props();
</script>

{#if type == "abstract.ACMEChallenge"}
    <ACMEChallengeEditor {dn} {origin} bind:value={value} />
{:else if type == "abstract.Delegation"}
    <DelegationEditor {dn} {origin} bind:value={value} />
{:else if type == "abstract.GithubOrgVerif"}
    <GithubOrgVerifEditor {dn} {origin} bind:value={value} />
{:else if type == "abstract.GitlabPageVerif"}
    <GitlabPageVerifEditor {dn} {origin} bind:value={value} />
{:else if type == "abstract.GoogleVerif"}
    <GoogleVerifEditor {dn} {origin} bind:value={value} />
{:else if type == "abstract.KeybaseVerif"}
    <KeybaseVerifEditor {dn} {origin} bind:value={value} />
{:else if type == "abstract.MatrixIM"}
    <MatrixIMEditor {dn} {origin} bind:value={value} />
{:else if type == "abstract.NSOnlyOrigin"}
    <NsOnlyOriginEditor {dn} {origin} bind:value={value} />
{:else if type == "abstract.OpenPGP"}
    <OpenPGPEditor {dn} {origin} bind:value={value} />
{:else if type == "abstract.Origin"}
    <OriginEditor {dn} {origin} bind:value={value} />
{:else if type == "abstract.RFC6186"}
    <RFC6186Editor {dn} {origin} bind:value={value} />
{:else if type == "abstract.ScalewayChallenge"}
    <ScalewayChallengeEditor {dn} {origin} bind:value={value} />
{:else if type == "abstract.Server"}
    <ServerEditor {dn} {origin} bind:value={value} />
{:else if type == "abstract.SMimeCert"}
    <SMimeCertEditor {dn} {origin} bind:value={value} />
{:else if type == "abstract.XMPP"}
    <XMPPEditor {dn} {origin} bind:value={value} />
{:else if type == "svcs.CAAPolicy"}
    <CAAEditor {dn} {origin} bind:value={value} />
{:else if type == "svcs.CNAME"}
    <CNAMEEditor {dn} {origin} bind:value={value} />
{:else if type == "svcs.DKIMRecord"}
    <DKIMEditor {dn} {origin} bind:value={value} />
{:else if type == "svcs.DMARC"}
    <DMARCEditor {dn} {origin} bind:value={value} />
{:else if type == "svcs.MTA_STS"}
    <MTA_STSEditor {dn} {origin} bind:value={value} />
{:else if type == "svcs.MXs"}
    <MXsEditor {dn} {origin} bind:value={value} />
{:else if type == "svcs.NAPTR"}
    <NAPTREditor {dn} {origin} bind:value={value} />
{:else if type == "svcs.PTR"}
    <PTREditor {dn} {origin} bind:value={value} />
{:else if type == "svcs.SPF"}
    <SPFEditor {dn} {origin} bind:value={value} />
{:else if type == "svcs.SSHFPs"}
    <SSHFPsEditor {dn} {origin} bind:value={value} />
{:else if type == "svcs.TLSAs"}
    <TLSAsEditor {dn} {origin} bind:value={value} />
{:else if type == "svcs.TLS_RPT"}
    <TLS_RPTEditor {dn} {origin} bind:value={value} />
{:else if type == "svcs.UnknownSRV"}
    <UnknownSRVEditor {dn} {origin} bind:value={value} />
{:else}
    <OrphanEditor
        {dn}
        {origin}
        {type}
        bind:value={value}
    />
{/if}
