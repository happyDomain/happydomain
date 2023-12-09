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
                    class="font-monospace text-truncate"
                    title={record.str}
                >
                    {record.str}
                </span>
            </div>
            {#if expand}
                <Row class="mt-2 flex-nowrap">
                    <Col>
                        <dl class="row ms-2">
                            <dt class="col-sm-5 text-end">
                                Class
                            </dt>
                            <dd class="col-sm-7 text-muted font-monospace mb-1">
                                {nsclass(record.rr.Hdr.Class)}
                            </dd>
                            <dt class="col-sm-5 text-end">
                                TTL
                            </dt>
                            <dd class="col-sm-7 text-muted font-monospace mb-1">
                                {record.rr.Hdr.Ttl}
                            </dd>
                            <dt class="col-sm-5 text-end">
                                RRType
                            </dt>
                            <dd class="col-sm-7 text-muted font-monospace mb-1">
                                {nsrrtype(record.rr.Hdr.Rrtype)}
                            </dd>
                        </dl>
                    </Col>
                    <Col sm="9">
                        <dl class="row">
                            {#each Object.keys(record.rr) as k}
                                {#if k != "Hdr"}
                                    {@const v = record.rr[k]}
                                    <dt class="col-sm-3 text-end">
                                        {k}
                                    </dt>
                                    <dd
                                        class="col-sm-9 text-muted font-monospace text-truncate mb-1"
                                        title={v}
                                    >
                                        {v}
                                    </dd>
                                {/if}
                            {/each}
                        </dl>
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
                    class="font-monospace"
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
            {:else if record.rr.Hdr.Rrtype != 6}
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
