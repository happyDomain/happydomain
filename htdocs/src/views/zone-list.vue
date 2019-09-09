<template>
  <div class="container mt-2">
    <h2>
      Zones
      <b-button v-b-modal.newZoneModal variant="primary" size="sm" class="float-right ml-2"><span class="glyphicon glyphicon-plus" aria-hidden="true"></span> Add zone</b-button>
    </h2>

    <table class="table table-hover table-bordered table-striped">
      <thead class="thead-dark">
        <tr>
          <th>
            Domain name
          </th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(zone, index) in zones" v-bind:key="index">
          <td @click="show(zone)" class="text-monospace">
            {{ zone }}
          </td>
          <td>
            <button type="button" @click="show(zone)" class="btn btn-sm btn-primary"><span class="glyphicon glyphicon-search" aria-hidden="true"></span></button>
            <button type="button" @click="deleteZone(zone)" class="btn btn-sm btn-danger"><span class="glyphicon glyphicon-trash" aria-hidden="true"></span></button>
          </td>
        </tr>
      </tbody>
    </table>

    <b-modal
      id="newZoneModal"
      ref="modal"
      title="Attach new zone"
      @show="resetModal"
      @shown="modalShown"
      @ok="handleOk"
    >
      <form ref="form" @submit.stop.prevent="handleSubmit">
        <b-form-group
          :state="domainNameState"
          label="Domain name"
          label-for="dn-input"
          invalid-feedback="Domain name is required"
        >
          <b-form-input
            id="dn-input"
            v-model="toAttachDN"
            :state="domainNameState"
            required
            placeholder="example.com"
            ref="domainname"
          ></b-form-input>
          <small id="dnHelp" class="form-text text-muted">Fill here the domain name you would like to manage with LibreDNS.</small>
        </b-form-group>
      </form>
    </b-modal>
  </div>
</template>

<script>
import axios from 'axios'

export default {

  data: function () {
    return {
      domainNameState: null,
      toAttachDN: '',
      zones: []
    }
  },

  mounted () {
    axios
      .get('/api/zones')
      .then(response => (this.zones = response.data))
  },

  methods: {
    attachZone (dn) {
      axios
        .post('/api/zones', {
          domainName: this.toAttachDN
        })
        .then(response => (this.zones.push(response.data.dn)))
    },

    deleteZone (dn) {
      axios
        .delete('/api/zones/' + dn)
        .then(response => (
          axios
            .get('/api/zones')
            .then(response => (this.zones = response.data))
        ))
    },

    show (dn) {
      this.$router.push('/zones/' + dn)
    },

    modalShown () {
      this.$refs.domainname.focus()
    },

    resetModal () {
      this.toAttachDN = ''
      this.domainNameState = null
    },

    checkFormValidity () {
      const valid = this.$refs.form.checkValidity()
      this.domainNameState = valid ? 'valid' : 'invalid'
      return valid
    },

    handleOk (bvModalEvt) {
      bvModalEvt.preventDefault()
      this.handleSubmit()
    },

    handleSubmit () {
      if (!this.checkFormValidity()) {
        return
      }

      this.attachZone(this.toAttachDN)

      this.$nextTick(() => {
        this.$refs.modal.hide()
      })
    }

  }
}
</script>
