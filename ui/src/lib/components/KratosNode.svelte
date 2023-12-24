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
 import {
     Button,
     FormGroup,
     Input,
     Label,
     Spinner,
 } from 'sveltestrap';

 import { t } from '$lib/translations';

 export let flow: String;
 export let i;
 export let node: Object;
 export let submissionInProgress = false;
 export let value: String;

 let isSocial = false;
 $: isSocial = node && node.attributes && (node.attributes.name === "provider" || node.attributes.name === "link") && node.group === "oidc"
</script>

{#if node.type === 'input'}
    {#if node.attributes.type === "hidden"}
        <input
            {...node.attributes}
        >
    {:else if node.attributes.type === "submit"}
        <FormGroup class="d-flex flex-column">
            <Button
                color="primary"
                {...node.attributes}
                disabled={submissionInProgress || node.attributes.disabled}
                formnovalidate={isSocial || node.meta.label.id === 107008}
            >
                {#if submissionInProgress}
                    <Spinner size="sm" />
                {/if}
                {node.meta.label.text}
            </Button>
        </FormGroup>
    {:else if node.attributes.type === "button"}
        <FormGroup class="d-flex flex-column">
            <Button
                type="button"
                color="secondary"
                name={node.attributes.name}
                value={node.attributes.value}
                disabled={node.attributes.disabled || submissionInProgress}
                on:click={(e) => {e.stopPropagation(); e.preventDefault(); const run = new Function(node.attributes.onclick); run();}}
            >
                {node.meta.label.text}
            </Button>
        </FormGroup>
    {:else if node.attributes.type === "checkbox"}
        <FormGroup>
            <Input
                type="checkbox"
                label={node.meta.label.text}
                id={"ns" + i}
                {...node.attributes}
                disabled={submissionInProgress || node.attributes.disabled}
                invalid={node.messages.find(({ type }) => type === "error")}
                feedback={node.messages.map(({ text, id }, k) => text)}
                on:changed={(e) => { if (e.target.checked) { value = node.attributes.value; } else { value = undefined; } }}
            />
        </FormGroup>
    {:else}
        <FormGroup>
            <Label for={"ns" + i}>
                {#if node.meta.label.id == 107001}
                    {$t('common.password')}
                {:else if node.meta.label.id == 107002 || node.meta.label.id == 107004}
                    {$t('email.address')}
                {:else}
                    {node.meta.label.text}
                {/if}
            </Label>
            <Input
                id={"ns" + i}
                {...node.attributes}
                disabled={submissionInProgress || node.attributes.disabled}
                invalid={node.messages.find(({ type }) => type === "error")}
                feedback={node.messages.map(({ text, id }, k) => text)}
                placeholder={node.attributes.placeholder?node.attributes.placeholder:(node.meta.label.id===107001?"pMockapetris@usc.edu":(node.meta.label.id===107002?"xXxXxXxXxX":""))}
                bind:value={value}
            />
            {#if flow === "login" && node.attributes.type === "password"}
                <div class="form-text">
                    <a
                        href="/forgotten-password"
                    >
                        {$t('password.forgotten')}
                    </a>
                </div>
            {/if}
        </FormGroup>
    {/if}
{:else if node.type === 'a'}
    <Button
        href={node.attributes.href}
    >
        {node.attributes.title.text}
    </Button>
{:else if node.type === 'img'}
    <img
        src={node.attributes.src}
        alt={node.meta.label?.text}
    />
{:else if node.type === 'script'}
    <script {...node.attributes}></script>
{:else if node.type === 'text'}
    <p>
        {node.meta?.label?.text}
    </p>
    {#if node.attributes.text.id === 1050015}
        <div
            class="container-fluid"
        >
            <div class="row">
                {#each node.attributes.text.context.secrets as text, k}
                    <div
                        key={k}
                        class="col-xs-3"
                    >
                        <code>{text.id === 1050014 ? "Used" : text.text}</code>
                    </div>
                {/each}
            </div>
        </div>
                    {:else}
        <div>
            <pre>
                {node.attributes.text.text}
            </pre>
        </div>
    {/if}
{/if}
