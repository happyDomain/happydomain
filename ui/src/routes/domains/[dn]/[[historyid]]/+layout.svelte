<script lang="ts">
 import { tick } from 'svelte';
 import { goto } from '$app/navigation';

 // @ts-ignore
 import { escape } from 'html-escaper';
 import {
     Alert,
     Button,
     ButtonGroup,
     Col,
     Container,
     Icon,
     Input,
     Modal,
     ModalBody,
     ModalFooter,
     ModalHeader,
     Row,
     Spinner,
     TabContent,
     TabPane,
 } from 'sveltestrap';

 import {
     getDomain as APIGetDomain,
     deleteDomain as APIDeleteDomain,
 } from '$lib/api/domains';
 import {
     applyZone as APIApplyZone,
     diffZone as APIDiffZone,
     importZone as APIImportZone,
     retrieveZone as APIRetrieveZone,
     viewZone as APIViewZone,
 } from '$lib/api/zone';
 import ImgProvider from '$lib/components/providers/ImgProvider.svelte';
 import type { Domain, DomainInList } from '$lib/model/domain';
 import type { ZoneMeta } from '$lib/model/zone';
 import { domains, domains_idx, refreshDomains } from '$lib/stores/domains';
 import { providers, providers_idx, refreshProviders } from '$lib/stores/providers';
 import { t } from '$lib/translations';

 export let data: {domain: string; history: string;};

 let selectedDomain = data.domain;
 $: if (selectedDomain != data.domain) {
     main_error = null;
     goto('/domains/' + encodeURIComponent(selectedDomain));
 }

 if (!$domains) refreshDomains();
 if (!$providers) refreshProviders();

 let domainsByGroup: Record<string, Array<DomainInList>> = {};
 $: {
     if ($domains) {
         const tmp: Record<string, Array<DomainInList>> = { };

         for (const domain of $domains) {
             if (tmp[domain.group] === undefined) {
                 tmp[domain.group] = [];
             }

             tmp[domain.group].push(domain);
         }

         domainsByGroup = tmp;
     }
 }

 let main_error: string | null = null;

 let selectedHistory: string | undefined = data.history;
 $: if (!data.history && $domains_idx[selectedDomain] && $domains_idx[selectedDomain].zone_history && $domains_idx[selectedDomain].zone_history.length > 0) {
     selectedHistory = $domains_idx[selectedDomain].zone_history[0] as string;
 }
 $: if (selectedHistory && data.history != selectedHistory) {
     main_error = null;
     goto('/domains/' + encodeURIComponent(selectedDomain) + '/' + encodeURIComponent(selectedHistory));
 }

 let retrievalInProgress = false;
 function retrieveZone(): void {
     if (domain) {
         retrievalInProgress = true;
         APIRetrieveZone(domain).then(
             retrieveZoneDone,
             (err: any) => {
                 retrievalInProgress = false;
                 throw err;
             }
         );
     }
 }

 function retrieveZoneDone(zm: ZoneMeta): void {
     uploadModalIsOpen = false;
     retrievalInProgress = false;
     uploadInProgress = false;
     refreshDomains();
     selectedHistory = zm.id;
     main_error = null;
 }

 async function getDomain(id: string): Promise<Domain> {
     return await APIGetDomain(id);
 }

 let domain: null | Domain = null;
 $: if ($domains_idx[selectedDomain]) {
     if (!$domains_idx[selectedDomain].zone_history || $domains_idx[selectedDomain].zone_history.length == 0) {
         retrievalInProgress = true;
         APIRetrieveZone($domains_idx[selectedDomain]).then(
             retrieveZoneDone,
             (err: any) => {
                 retrievalInProgress = false;

                 tick().then(() => {
                     main_error = err.toString();
                 });
             }
         )
     } else {
         domain = null;
         getDomain($domains_idx[selectedDomain].id).then(
             (dn) => {
                 domain = dn;
             }
         );
     }
 }

 let uploadModalIsOpen = false;
 let uploadInProgress = false;
 let zoneImportContent = "";
 let zoneImportFiles: FileList;
 let uploadModalActiveTab: string|number = 0;

 $: if (uploadModalIsOpen) {
     uploadInProgress = false;
     zoneImportContent = "";
     uploadModalActiveTab = 0;
 }

 function importZone(): void {
     if (domain && selectedHistory) {
         uploadInProgress = true;
         let file = new Blob([zoneImportContent], {"type": "text/plain"});
         if (uploadModalActiveTab != "uploadText") {
             file = zoneImportFiles[0];
         }
         APIImportZone(domain, selectedHistory, file).then(
             retrieveZoneDone,
             (err: any) => {
                 uploadInProgress = false;
                 throw err;
             }
         );
     }
 }

 let viewZoneModalIsOpen = false;
 let zoneContent: null | string = null;
 function viewZone(): void {
     if (domain && selectedHistory) {
         zoneContent = null;
         viewZoneModalIsOpen = true;
         APIViewZone(domain, selectedHistory).then(
             (v: string) => zoneContent = v,
             (err: any) => {
                 viewZoneModalIsOpen = false;
                 throw err;
             }
         );
     }
 }

 let applyZoneModalIsOpen = false;
 let zoneDiff: Array<{className: string; msg: string;}> | null = null;
 let zoneDiffCreated = 0;
 let zoneDiffDeleted = 0;
 let zoneDiffModified = 0;
 function showDiff(): void {
     if (!domain || !selectedHistory) {
         return;
     }

     zoneDiff = null;
     selectedDiff = null;
     applyZoneModalIsOpen = true;
     propagationInProgress = false;
     APIDiffZone(domain, '@', selectedHistory).then(
         (v: Array<string>) => {
             zoneDiffCreated = 0;
             zoneDiffDeleted = 0;
             zoneDiffModified = 0;
             if (v) {
                 zoneDiff = v.map(
                     (msg: string) => {
                         let className = '';
                         if (/^± MODIFY/.test(msg)) {
                             className = 'text-warning';
                             zoneDiffModified += 1;
                         } else if (/^\+ CREATE/.test(msg)) {
                             className = 'text-success';
                             zoneDiffCreated += 1;
                         } else if (/^- DELETE/.test(msg)) {
                             className = 'text-danger';
                             zoneDiffDeleted += 1;
                         } else if (/^REFRESH/.test(msg)) {
                             className = 'text-info';
                         }

                         return {
                             className,
                             msg,
                         };
                     }
                 );
             } else {
                 zoneDiff = [];
             }
             selectedDiff = v;
         },
         (err: any) => {
             applyZoneModalIsOpen = false;
             throw err;
         }
     )
 }

 let selectedDiff: Array<string> | null = null;
 let selectedDiffCreated = 0;
 let selectedDiffDeleted = 0;
 let selectedDiffModified = 0;
 $: selectedDiffCreated = !selectedDiff?0:selectedDiff.filter((msg: string) => /^\+ CREATE/.test(msg)).length;
 $: selectedDiffDeleted = !selectedDiff?0:selectedDiff.filter((msg: string) => /^- DELETE/.test(msg)).length;
 $: selectedDiffModified = !selectedDiff?0:selectedDiff.filter((msg: string) => /^± MODIFY/.test(msg)).length;

 let propagationInProgress = false;
 async function applyDiff() {
     if (!domain || !selectedHistory || !selectedDiff) return;

     propagationInProgress = true;
     try {
         retrieveZoneDone(await APIApplyZone(domain, selectedHistory, selectedDiff));
     } finally {
         applyZoneModalIsOpen = false;
     }
 }

 let deleteModalIsOpen = false;
 let deleteInProgress = false;
 function detachDomain(): void {
     deleteInProgress = true;
     APIDeleteDomain($domains_idx[selectedDomain].id).then(
         () => {
             refreshDomains().then(() => {
                 deleteInProgress = false;
                 goto('/domains');
             }, () => {
                 deleteInProgress = false;
                 goto('/domains');
             });
         },
         (err: any) => {
             deleteInProgress = false;
             throw err;
         }
     );
 }
</script>

<Container
    fluid
    class="d-flex flex-column flex-fill"
>
    <Row
        class="flex-fill"
    >
        <Col
            sm={4}
            md={3}
            class="bg-light py-2 sticky-top d-flex flex-column justify-content-between"
            style="overflow-y: auto; max-height: 100vh; z-index: 0"
        >
            {#if $domains_idx[selectedDomain]}
                <div class="d-flex">
                    <Button href="/domains/" class="fw-bolder" color="link">
                        <Icon name="chevron-up" />
                    </Button>
                    <Input
                        type="select"
                        bind:value={selectedDomain}
                    >
                    {#each Object.keys(domainsByGroup) as gname}
                        {@const group = domainsByGroup[gname]}
                        <optgroup label={gname=="undefined"?$t("domaingroups.no-group"):gname}>
                            {#each group as domain}
                                <option value={domain.domain}>{domain.domain}</option>
                            {/each}
                        </optgroup>
                    {/each}
                </Input>
                </div>

                {#if domain && domain.zone_history && $domains_idx[selectedDomain] && domain.id === $domains_idx[selectedDomain].id}
                    <form class="mt-3">
                        <div class="d-flex justify-content-between">
                            <label class="fw-bolder" for="zhistory">{$t('domains.history')}:</label>
                            <div class="d-flex gap-1">
                                <Button
                                    outline
                                    color="secondary"
                                    size="sm"
                                    title={$t('domains.actions.upload')}
                                    disabled={uploadInProgress}
                                    on:click={() => uploadModalIsOpen = true}
                                >
                                    {#if uploadInProgress}
                                        <Spinner size="sm" />
                                    {:else}
                                        <Icon name="cloud-upload" />
                                    {/if}
                                </Button>
                                <Button
                                    outline
                                    color="secondary"
                                    size="sm"
                                    title={$t('domains.actions.reimport')}
                                    disabled={retrievalInProgress}
                                    on:click={retrieveZone}
                                >
                                    {#if retrievalInProgress}
                                        <Spinner size="sm" />
                                    {:else}
                                        <Icon name="cloud-download" />
                                    {/if}
                                </Button>
                            </div>
                        </div>
                        {#key domain.zone_history}
                            <select
                                class="form-select"
                                id="zhistory"
                                bind:value={selectedHistory}
                            >
                                {#each domain.zone_history as history}
                                    <option value={history.id}>{history.last_modified}</option>
                                {/each}
                            </select>
                        {/key}
                    </form>

                    <ButtonGroup class="mt-3 w-100">
                        <Button
                            size="sm"
                            outline
                            color="info"
                            title={$t('domains.actions.view')}
                            on:click={viewZone}
                        >
                            <Icon name="list-ul" aria-hidden="true" /><br>
                            {$t('domains.actions.view')}
                        </Button>
                        {#if $domains_idx[selectedDomain].zone_history && selectedHistory === $domains_idx[selectedDomain].zone_history[0]}
                            <Button
                                size="sm"
                                color="success"
                                title={$t('domains.actions.propagate')}
                                on:click={showDiff}
                            >
                                <Icon name="cloud-upload" aria-hidden="true" /><br>
                                {$t('domains.actions.propagate')}
                            </Button>
                        {:else}
                            <Button
                                size="sm"
                                color="warning"
                                title={$t('domains.actions.rollback')}
                                on:click={showDiff}
                            >
                                <Icon name="cloud-upload" aria-hidden="true" /><br>
                                {$t('domains.actions.rollback')}
                            </Button>
                        {/if}
                    </ButtonGroup>
                {/if}

                <div class="flex-fill my-3" />

                <Button
                    class="w-100"
                    type="button"
                    outline
                    color="danger"
                    disabled={deleteInProgress}
                    on:click={() => deleteModalIsOpen = true}
                >
                    {#if deleteInProgress}
                        <Spinner size="sm" />
                    {:else}
                        <Icon name="trash-fill" />
                    {/if}
                    {$t('domains.stop')}
                </Button>

                {#if $providers_idx && $providers_idx[$domains_idx[selectedDomain].id_provider]}
                    <form class="mt-2">
                        <!-- svelte-ignore a11y-label-has-associated-control -->
                        <label class="font-weight-bolder">
                            {$t('domains.view.provider')}:
                        </label>
                        <div class="pr-2 pl-2">
                            <Button
                                href={"/providers/" + encodeURIComponent($domains_idx[selectedDomain].id_provider)}
                                class="p-3 w-100 text-left"
                                type="button"
                                color="info"
                                outline
                            >
                                <div
                                    class="d-inline-block text-center"
                                    style="width: 50px;"
                                >
                                    <ImgProvider id_provider={$domains_idx[selectedDomain].id_provider} />
                                </div>
                                {$providers_idx[$domains_idx[selectedDomain].id_provider]._comment}
                            </Button>
                        </div>
                    </form>
                {/if}
            {:else}
                <div class="mt-4 text-center">
                    <Spinner color="primary" />
                </div>
            {/if}
        </Col>
        <Col
            sm={8}
            md={9}
            class="d-flex"
        >
            {#if main_error}
                <div class="d-flex flex-column mt-3">
                    <Alert
                        color="danger"
                        fade={false}
                    >
                        <strong>{$t('errors.domain-import')}</strong>
                        {main_error}
                    </Alert>
                </div>
            {:else if data.history == selectedHistory}
                <slot />
            {:else}
                <div class="mt-5 text-center flex-fill">
                    <Spinner label="Spinning" />
                    <p>{$t('wait.loading')}</p>
                </div>
            {/if}
        </Col>
    </Row>
</Container>

<Modal
    isOpen={uploadModalIsOpen}
    size="lg"
>
    <ModalHeader toggle={() => uploadModalIsOpen = false}>{$t('zones.upload')}</ModalHeader>
    <ModalBody>
        <TabContent on:tab={(e) => (uploadModalActiveTab = e.detail)}>
            <TabPane tabId="uploadText" tab={$t('zones.import-text')} active>
                <Input
                    class="mt-3"
                    type="textarea"
                    style="height: 200px;"
                    placeholder="@         4269 IN SOA   root ns 2042070136 ..."
                    bind:value={zoneImportContent}
                />
            </TabPane>
            <TabPane tabId="uploadFile" tab={$t('zones.import-file')}>
                {#if uploadModalIsOpen}
                    <Input
                        class="mt-3"
                        type="file"
                        bind:files={zoneImportFiles}
                    />
                {/if}
            </TabPane>
        </TabContent>
    </ModalBody>
    <ModalFooter>
        <Button
            outline
            color="secondary"
            on:click={() => uploadModalIsOpen = false}
        >
            {$t('common.cancel')}
        </Button>
        <Button
            color="primary"
            disabled={uploadInProgress}
            on:click={importZone}
        >
            {#if uploadInProgress}
                <Spinner size="sm" />
            {/if}
            {$t('domains.actions.upload')}
        </Button>
    </ModalFooter>
</Modal>

<Modal
    isOpen={deleteModalIsOpen}
    size="lg"
>
    <ModalHeader toggle={() => deleteModalIsOpen = false}>{$t('domains.removal')}</ModalHeader>
    <ModalBody>
        {$t('domains.alert.remove')}
    </ModalBody>
    <ModalFooter>
        <Button
            outline
            color="secondary"
            on:click={() => deleteModalIsOpen = false}
        >
            {$t('domains.view.cancel-title')}
        </Button>
        <Button
            color="danger"
            on:click={detachDomain}
        >
            {$t('domains.discard')}
        </Button>
    </ModalFooter>
</Modal>

<Modal
    isOpen={viewZoneModalIsOpen}
    size="lg"
    scrollable
>
    <ModalHeader toggle={() => viewZoneModalIsOpen = false}>{$t('domains.view.title')}</ModalHeader>
    <ModalBody>
        {#if zoneContent}
            <pre style="overflow: initial">{zoneContent}</pre>
        {:else}
            <div class="my-2 text-center">
                <Spinner label="Spinning" />
                <p>{$t('wait.formating')}</p>
            </div>
        {/if}
    </ModalBody>
</Modal>

<Modal
    isOpen={applyZoneModalIsOpen}
    size="lg"
    scrollable
>
    {#if domain}
        <ModalHeader toggle={() => applyZoneModalIsOpen = false}>
            {@html $t('domains.view.description', {"domain": `<span class="font-monospace">${escape(domain.domain)}</span>`})}
        </ModalHeader>
    {/if}
    <ModalBody>
        {#if !zoneDiff}
            <div class="my-2 text-center">
                <Spinner color="warning" label="Spinning" />
                <p>{$t('wait.exporting')}</p>
            </div>
        {:else if zoneDiff.length == 0}
            <div class="d-flex gap-3 align-items-center justify-content-center">
                <Icon name="check2-all" class="display-5 text-success" />
                {$t('domains.apply.nochange')}
            </div>
        {:else}
            {#each zoneDiff as line, n}
                <div
                    class={'col font-monospace form-check ' + line.className}
                >
                    <input
                        type="checkbox"
                        class="form-check-input"
                        id="zdiff{n}"
                        bind:group={selectedDiff}
                        value={line.msg}
                    />
                    <label
                        class="form-check-label"
                        for="zdiff{n}"
                        style="padding-left: 1em; text-indent: -1em;"
                    >
                        {line.msg}
                    </label>
                </div>
            {/each}
        {/if}
    </ModalBody>
    <ModalFooter>
        {#if zoneDiff}
            {#if zoneDiffCreated}
                <span class="text-success">
                    {$t('domains.apply.additions', {count: selectedDiffCreated})}
                </span>
            {/if}
            {#if zoneDiffCreated && zoneDiffDeleted}
                &ndash;
            {/if}
            {#if zoneDiffDeleted}
                <span class="text-danger">
                    {$t('domains.apply.deletions', {count: selectedDiffDeleted})}
                </span>
            {/if}
            {#if (zoneDiffCreated || zoneDiffDeleted) && zoneDiffModified}
                &ndash;
            {/if}
            {#if zoneDiffModified}
                <span class="text-warning">
                    {$t('domains.apply.modifications', {count: selectedDiffModified})}
                </span>
            {/if}
            {#if (zoneDiffCreated || zoneDiffDeleted || zoneDiffModified) && (zoneDiff.length - zoneDiffCreated - zoneDiffDeleted - zoneDiffModified !== 0)}
                &ndash;
            {/if}
            {#if selectedDiff && zoneDiff.length - zoneDiffCreated - zoneDiffDeleted - zoneDiffModified !== 0}
                <span class="text-info">
                    {$t('domains.apply.others', {count: selectedDiff.length - selectedDiffCreated - selectedDiffDeleted - selectedDiffModified})}
                </span>
            {/if}
        {/if}
        <div class="d-flex gap-1">
            <Button outline color="secondary" on:click={() => applyZoneModalIsOpen = false}>
                {$t('common.cancel')}
            </Button>
            <Button color="success" disabled={propagationInProgress || !zoneDiff || !selectedDiff || selectedDiff.length === 0} on:click={applyDiff}>
                {#if propagationInProgress}
                    <Spinner label="Spinning" size="sm" />
                {/if}
                {$t('domains.apply.button')}
            </Button>
        </div>
    </ModalFooter>
</Modal>
