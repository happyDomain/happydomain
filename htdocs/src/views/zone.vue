<template>
  <div class="container mt-2">
    <h2>{{ zone.dn }}</h2>
    <div v-show="zone">
      <h3>
        Resource Record ({{ rrs.length }})
        <button type="button" @click="addRR()" class="float-right btn btn-sm btn-primary ml-2"><span class="glyphicon glyphicon-plus" aria-hidden="true"></span> Add RR</button>
      </h3>

      <table class="table table-hover table-bordered table-striped table-sm" style="table-layout: fixed;">
        <thead>
          <tr>
            <th>resource records</th>
            <th style="width: 5%;">act</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(rr, index) in rrs" v-bind:key="index">
            <td v-if="!rr.edit" style="overflow:hidden; text-overflow: ellipsis;white-space: nowrap;">
              <span @click="toogleRR(index)" class="glyphicon glyphicon-chevron-right" v-if="!rr.expand"></span>
              <span @click="toogleRR(index)" class="glyphicon glyphicon-chevron-down" v-if="rr.expand"></span>
              <span @click="toogleRR(index)" class="text-monospace" :title="rr.string">{{ rr.string }}</span>
              <div class="row" v-show="rr.expand">
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
                <input class="form-control text-monospace" v-model="rr.string">
              </form>
            </td>
            <td>
              <button type="button" @click="deleteRR(index)" class="btn btn-sm btn-danger" v-if="!rr.edit && rr.fields.Hdr.Rrtype != 6"><span class="glyphicon glyphicon-trash" aria-hidden="true"></span></button>
              <button type="button" @click="newRR(index)" class="btn btn-sm btn-success" v-if="rr.edit"><span class="glyphicon glyphicon-ok" aria-hidden="true"></span></button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script>
import axios from 'axios'
import Vue from 'vue'

export default {

  data: function () {
    return {
      rrs: {},
      query: '',
      zone: {}
    }
  },

  mounted () {
    var myzone = this.$route.params.zone
    axios
      .get('/api/zones/' + myzone)
      .then(response => (this.zone = response.data))

    axios
      .get('/api/zones/' + myzone + '/rr')
      .then(response => (this.rrs = response.data))
  },

  methods: {
    toogleRR (idx) {
      var tmp = this.rrs[idx]
      tmp.expand = !tmp.expand
      Vue.set(this.rrs, idx, tmp)
    },

    addRR () {
      this.rrs.push({ edit: true })
    },

    newRR (idx) {
      axios
        .post('/api/zones/' + this.$route.params.zone + '/rr', {
          'string': this.rrs[idx].string
        })
        .then(
          (response) => {
            axios
              .get('/api/zones/' + this.$route.params.zone + '/rr')
              .then(response => (this.rrs = response.data))
          },
          (error) => {
            alert('An error occurs when trying to add RR to zone: ' + error.response.data.errmsg)
          }
        )
    },

    deleteRR (idx) {
      axios
        .delete('/api/zones/' + this.$route.params.zone + '/rr', { data: {
          'string': this.rrs[idx].string
        } })
        .then(
          (response) => {
            axios
              .get('/api/zones/' + this.$route.params.zone + '/rr')
              .then(response => (this.rrs = response.data))
          },
          (error) => {
            alert('An error occurs when trying to delete RR in zone: ' + error.response.data.errmsg)
          }
        )
    }
  }
}
</script>
