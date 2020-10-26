<!--
    Copyright or © or Copr. happyDNS (2020)

    contact@happydns.org

    This software is a computer program whose purpose is to provide a modern
    interface to interact with DNS systems.

    This software is governed by the CeCILL license under French law and abiding
    by the rules of distribution of free software.  You can use, modify and/or
    redistribute the software under the terms of the CeCILL license as
    circulated by CEA, CNRS and INRIA at the following URL
    "http://www.cecill.info".

    As a counterpart to the access to the source code and rights to copy, modify
    and redistribute granted by the license, users are provided only with a
    limited warranty and the software's author, the holder of the economic
    rights, and the successive licensors have only limited liability.

    In this respect, the user's attention is drawn to the risks associated with
    loading, using, modifying and/or developing or reproducing the software by
    the user in light of its specific status of free software, that may mean
    that it is complicated to manipulate, and that also therefore means that it
    is reserved for developers and experienced professionals having in-depth
    computer knowledge. Users are therefore encouraged to load and test the
    software's suitability as regards their requirements in conditions enabling
    the security of their systems and/or data to be ensured and, more generally,
    to use and operate it in the same conditions as regards security.

    The fact that you are presently reading this means that you have had
    knowledge of the CeCILL license and that you accept its terms.
  -->

<template>
  <tr>
    <td v-if="!record.edit" style="overflow:hidden; text-overflow: ellipsis;white-space: nowrap;">
      <b-icon v-if="!expand" icon="chevron-right" @click="toogleRR()" />
      <b-icon v-if="expand" icon="chevron-down" @click="toogleRR()" />
      <span class="text-monospace" :title="record.string" @click="toogleRR()">{{ record.string }}</span>
      <div v-show="expand" class="row">
        <dl class="col-sm-6 row">
          <dt class="col-sm-3 text-right">
            Class
          </dt>
          <dd class="col-sm-9 text-muted text-monospace">
            {{ record.fields.Hdr.Class | nsclass }}
          </dd>
          <dt class="col-sm-3 text-right">
            TTL
          </dt>
          <dd class="col-sm-9 text-muted text-monospace">
            {{ record.fields.Hdr.Ttl }}
          </dd>
          <dt class="col-sm-3 text-right">
            RRType
          </dt>
          <dd class="col-sm-9 text-muted text-monospace">
            {{ record.fields.Hdr.Rrtype | nsrrtype }}
          </dd>
        </dl>
        <ul class="col-sm-6" style="list-style: none">
          <li v-for="(v,k) in record.fields" :key="k">
            <strong class="float-left mr-2">{{ k }}</strong> <span class="text-muted text-monospace" style="display:block;overflow:hidden; text-overflow: ellipsis;white-space: nowrap;" :title="v">{{ v }}</span>
          </li>
        </ul>
      </div>
    </td>
    <td v-if="record.edit && actBtn">
      <form @submit.stop.prevent="$emit('save-rr')">
        <input autofocus class="form-control text-monospace" :value="record.string" @input="r = record; r.string = $event; $emit('input', r)">
      </form>
    </td>
    <td v-if="actBtn">
      <button v-if="!record.edit && record.fields.Hdr.Rrtype != 6" type="button" class="btn btn-sm btn-danger" @click="$emit('delete-rr')">
        <b-icon icon="trash-fill" aria-hidden="true" />
      </button>
      <button v-if="record.edit" type="button" class="btn btn-sm btn-success" @click="$emit('save-rr')">
        <b-icon icon="check" aria-hidden="true" />
      </button>
    </td>
  </tr>
</template>

<script>
export default {
  name: 'HRecord',

  props: {
    actBtn: {
      type: Boolean,
      default: false
    },
    record: {
      type: Object,
      required: true
    }
  },

  data () {
    return {
      expand: false
    }
  },

  methods: {
    toogleRR () {
      this.expand = !this.expand
    }

  }

}
</script>
