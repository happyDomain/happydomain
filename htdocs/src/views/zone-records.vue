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
  <div class="container mt-2">
    <h2>
      <b-button v-show="rrs && rrs.length" class="float-right ml-2" size="sm" variant="primary" @click="addRR()">
        <b-icon icon="plus" aria-hidden="true" /> Add RR
      </b-button>
      <b-button-group v-show="rrs && rrs.length" class="float-right ml-2" size="sm" variant="secondary">
        <b-button :pressed.sync="showDNSSEC">
          DNSSEC
        </b-button>
      </b-button-group>
      <router-link :to="'/domains/' + domain.domain" class="btn">
        <b-icon icon="chevron-left" />
      </router-link>
      <span class="text-monospace">{{ domain.domain }}</span>
      <small class="text-muted">Resource Records <span v-if="rrs && rrs.length">({{ rrsFiltered.length }}/{{ rrs.length }})</span></small>
    </h2>
    <b-alert variant="danger" :show="error.length != 0">
      <strong>Error:</strong> {{ error }}
    </b-alert>
    <div v-show="rrs.length">
      <table class="table table-hover table-bordered table-striped table-sm" style="table-layout: fixed;">
        <thead>
          <tr>
            <th>resource records</th>
            <th style="width: 5%;">
              act
            </th>
          </tr>
        </thead>
        <tbody>
          <h-record v-for="(rr, index) in rrsFiltered" act-btn :record="rr" :key="index" @save-rr="newRR(index)" @delete-rr="deleteRR(index)" />
        </tbody>
      </table>
    </div>
    <div v-if="!rrs.length && error.length == 0" class="text-center mt-5">
      <b-spinner :label="$t('common.spinning')" />
      <p>{{ $t('wait.loading-record') }}</p>
    </div>
  </div>
</template>

<script>
import axios from 'axios'
import Vue from 'vue'

export default {

  components: {
    hRecord: () => import('@/components/hRecord')
  },

  data: function () {
    return {
      showDNSSEC: false,
      error: '',
      rrs: [],
      query: '',
      domain: {}
    }
  },

  computed: {
    rrsFiltered: function () {
      var ret = []
      if (this.rrs === null) {
        return ret
      }
      for (var k in this.rrs) {
        if (this.showDNSSEC || this.rrs[k].fields === undefined || (this.rrs[k].fields.Hdr.Rrtype !== 46 && this.rrs[k].fields.Hdr.Rrtype !== 47 && this.rrs[k].fields.Hdr.Rrtype !== 50)) {
          ret.push(this.rrs[k])
        }
      }
      return ret
    }
  },

  mounted () {
    var mydomain = this.$route.params.domain
    axios
      .get('/api/domains/' + encodeURIComponent(mydomain))
      .then(response => (this.domain = response.data))

    axios
      .get('/api/domains/' + encodeURIComponent(mydomain) + '/rr')
      .then(
        (response) => (this.rrs = response.data),
        (error) => (this.error = error.response.data.errmsg)
      )
  },

  methods: {
    addRR () {
      this.rrs.push({ edit: true })
      window.scrollTo(0, document.body.scrollHeight)
    },

    newRR (idx) {
      axios
        .post('/api/domains/' + encodeURIComponent(this.$route.params.domain) + '/rr', {
          string: this.rrsFiltered[idx].string
        })
        .then(
          (response) => {
            axios
              .get('/api/domains/' + encodeURIComponent(this.$route.params.domain) + '/rr')
              .then(response => (this.rrs = response.data))
          },
          (error) => {
            alert(this.$t('errors.rr-add') + ' ' + error.response.data.errmsg)
          }
        )
    },

    deleteRR (idx) {
      axios
        .delete('/api/domains/' + encodeURIComponent(this.$route.params.domain) + '/rr', {
          data: {
            string: this.rrsFiltered[idx].string
          }
        })
        .then(
          (response) => {
            axios
              .get('/api/domains/' + encodeURIComponent(this.$route.params.domain) + '/rr')
              .then(response => (this.rrs = response.data))
          },
          (error) => {
            alert(this.$t('errors.rr-delete') + ' ' + error.response.data.errmsg)
          }
        )
    }
  }
}
</script>
