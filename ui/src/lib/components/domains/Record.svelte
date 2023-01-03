<script lang="ts">
 import {
     Button,
     Col,
     Icon,
     Input,
     Row,
 } from 'sveltestrap';

 import { nsclass, nsrrtype } from '$lib/dns';
 import type { ServiceRecord } from '$lib/model/zone';

 export let actBtn = false;
 export let expand: boolean = false;
 export let record: ServiceRecord;
</script>

<tr>
    {#if !record.edit}
        <td style="max-width: 0;">
            <div
                class="d-flex"
                on:click={() => {expand = !expand}}
                on:keypress={() => {expand = !expand}}
            >
                {#if expand}
                    <Icon name="chevron-down" />
                {:else}
                    <Icon name="chevron-right" />
                {/if}
                <span
                    class="text-monospace text-truncate"
                    title={record.str}
                >
                    {record.str}
                </span>
            </div>
            {#if expand}
                <Row class="mt-2 flex-nowrap">
                    <Col>
                        <dl class="row">
                            <dt class="col-sm-3 text-end">
                                Class
                            </dt>
                            <dd class="col-sm-9 text-muted text-monospace">
                                {nsclass(record.fields.Hdr.Class)}
                            </dd>
                            <dt class="col-sm-3 text-end">
                                TTL
                            </dt>
                            <dd class="col-sm-9 text-muted text-monospace">
                                {record.fields.Hdr.Ttl}
                            </dd>
                            <dt class="col-sm-3 text-end">
                                RRType
                            </dt>
                            <dd class="col-sm-9 text-muted text-monospace">
                                {nsrrtype(record.fields.Hdr.Rrtype)}
                            </dd>
                        </dl>
                    </Col>
                    <Col style="max-width:60%">
                        <ul style="list-style: none">
                            {#each Object.keys(record.fields) as k}
                                {#if k != "Hdr"}
                                    {@const v = record.fields[k]}
                                    <li class="d-flex">
                                        <strong class="float-start me-2">{k}</strong>
                                        <div
                                            class="text-muted text-monospace text-truncate"
                                            title={v}
                                        >
                                            {v}
                                        </div>
                                    </li>
                                {/if}
                            {/each}
                        </ul>
                    </Col>
                </Row>
            {/if}
        </td>
    {:else}
        <td>
            <form
                submit="$emit('save-rr')"
            >
                <Input
                    autofocus
                    class="text-monospace"
                    bsSize="sm"
                    bind:value={record.str}
                />
            </form>
        </td>
    {/if}
    {#if record.edit || actBtn}
        <td>
            {#if record.edit}
                <Button
                    size="sm"
                    color="success"
                    click="$emit('save-rr')"
                >
                    <Icon name="check" aria-hidden="true" />
                </Button>
            {:else if record.fields.Hdr.Rrtype != 6}
                <Button
                    size="sm"
                    color="danger"
                    click="$emit('delete-rr')"
                >
                    <Icon name="trash-fill" aria-hidden="true" />
                </Button>
            {/if}
        </td>
    {/if}
</tr>
