<template>
  <div class="container mt-2">
    <h2>
      <b-button @click="addRR()" class="float-right ml-2" size="sm" variant="primary" v-show="rrs && rrs.length"><b-icon icon="plus" aria-hidden="true"></b-icon> Add RR</b-button>
      <b-button-group v-show="rrs && rrs.length" class="float-right ml-2" size="sm" variant="secondary">
        <b-button :pressed.sync="showDNSSEC">DNSSEC</b-button>
      </b-button-group>
      <router-link :to="'/domains/' + domain.domain" class="btn"><b-icon icon="chevron-left"></b-icon></router-link>
      {{ domain.domain }}
      <small class="text-muted">Resource Records <span v-if="rrs && rrs.length">({{ rrsFiltered.length }}/{{ rrs.length }})</span></small>
    </h2>
    <b-alert variant="danger" :show="error.length != 0"><strong>Error:</strong> {{ error }}</b-alert>
    <div v-show="rrs.length">
      <table class="table table-hover table-bordered table-striped table-sm" style="table-layout: fixed;">
        <thead>
          <tr>
            <th>resource records</th>
            <th style="width: 5%;">act</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(rr, index) in rrsFiltered" v-bind:key="index">
            <td v-if="!rr.edit" style="overflow:hidden; text-overflow: ellipsis;white-space: nowrap;">
              <b-icon @click="toogleRR(index)" icon="chevron-right" v-if="!expandrrs[index]"></b-icon>
              <b-icon @click="toogleRR(index)" icon="chevron-down" v-if="expandrrs[index]"></b-icon>
              <span @click="toogleRR(index)" class="text-monospace" :title="rr.string">{{ rr.string }}</span>
              <div class="row" v-show="expandrrs[index]">
                <dl class="col-sm-6 row">
                  <dt class="col-sm-3 text-right">Class</dt>
                  <dd class="col-sm-9 text-muted text-monospace">{{ rr.fields.Hdr.Class | nsclass }}</dd>
                  <dt class="col-sm-3 text-right">TTL</dt>
                  <dd class="col-sm-9 text-muted text-monospace">{{ rr.fields.Hdr.Ttl }}</dd>
                  <dt class="col-sm-3 text-right">RRType</dt>
                  <dd class="col-sm-9 text-muted text-monospace">{{ rr.fields.Hdr.Rrtype | nsrrtype }}</dd>
                </dl>
                <ul class="col-sm-6" style="list-style: none">
                  <li v-for="(v,k) in rr.fields" v-bind:key="k">
                    <strong class="float-left mr-2">{{ k }}</strong> <span class="text-muted text-monospace" style="display:block;overflow:hidden; text-overflow: ellipsis;white-space: nowrap;" :title="v">{{ v }}</span>
                  </li>
                </ul>
              </div>
            </td>
            <td v-if="rr.edit">
              <form @submit.stop.prevent="newRR(index)">
                <input autofocus class="form-control text-monospace" v-model="rr.string">
              </form>
            </td>
            <td>
              <button type="button" @click="deleteRR(index)" class="btn btn-sm btn-danger" v-if="!rr.edit && rr.fields.Hdr.Rrtype != 6"><b-icon icon="trash-fill" aria-hidden="true"></b-icon></button>
              <button type="button" @click="newRR(index)" class="btn btn-sm btn-success" v-if="rr.edit"><b-icon icon="check" aria-hidden="true"></b-icon></button>
            </td>
          </tr>
        </tbody>
      </table>

    </div>
    <div class="text-center mt-5" v-if="!rrs.length && error.length == 0">
      <b-spinner label="Spinning"></b-spinner>
      <p>Loading records&nbsp;&hellip;</p>
    </div>
  </div>
</template>

<script>
import axios from 'axios'
import Vue from 'vue'

export default {

  data: function () {
    return {
      showDNSSEC: false,
      error: '',
      expandrrs: {},
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
      .get('/api/domains/' + mydomain)
      .then(response => (this.domain = response.data))

    axios
      .get('/api/domains/' + mydomain + '/rr')
      .then(
        (response) => (this.rrs = response.data),
        (error) => (this.error = error.response.data.errmsg)
      )
  },

  methods: {
    toogleRR (idx) {
      Vue.set(this.expandrrs, idx, !this.expandrrs[idx])
    },

    addRR () {
      this.rrs.push({ edit: true })
      window.scrollTo(0, document.body.scrollHeight)
    },

    newRR (idx) {
      axios
        .post('/api/domains/' + this.$route.params.domain + '/rr', {
          string: this.rrsFiltered[idx].string
        })
        .then(
          (response) => {
            axios
              .get('/api/domains/' + this.$route.params.domain + '/rr')
              .then(response => (this.rrs = response.data))
          },
          (error) => {
            alert('An error occurs when trying to add RR to the zone: ' + error.response.data.errmsg)
          }
        )
    },

    deleteRR (idx) {
      axios
        .delete('/api/domains/' + this.$route.params.domain + '/rr', {
          data: {
            string: this.rrsFiltered[idx].string
          }
        })
        .then(
          (response) => {
            axios
              .get('/api/domains/' + this.$route.params.domain + '/rr')
              .then(response => (this.rrs = response.data))
          },
          (error) => {
            alert('An error occurs when trying to delete RR in the zone: ' + error.response.data.errmsg)
          }
        )
    }
  }
}
</script>
