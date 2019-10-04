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
            {{ zone.domain }}
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
          :state="newForm.domainNameState"
          label="Domain name"
          label-for="dn-input"
          invalid-feedback="Domain name is required"
        >
          <b-form-input
            id="dn-input"
            v-model="newForm.domain"
            :state="newForm.domainNameState"
            required
            placeholder="example.com"
            ref="domainname"
          ></b-form-input>
          <small id="dnHelp" class="form-text text-muted">Fill here the domain name you would like to manage with HappyDNS.</small>
        </b-form-group>
        <b-form-group
          :state="newForm.domainServerState"
          label="Server"
          label-for="srv-input"
        >
          <b-form-input
            id="srv-input"
            v-model="newForm.server"
            :state="newForm.domainServerState"
            placeholder="ns0.happydns.org"
            ref="domainserver"
          ></b-form-input>
        </b-form-group>
        <b-form-group
          :state="newForm.keyNameState"
          label="Dynamic DNS Update Key Name"
          label-for="keyname-input"
        >
          <b-form-input
            id="heyname-input"
            v-model="newForm.keyname"
            :state="newForm.keynameState"
            required
            placeholder="foo"
            ref="keyname"
          ></b-form-input>
        </b-form-group>
        <b-form-group
          :state="newForm.keyBlobState"
          label="Dynamic DNS Update Key"
          label-for="keyblob-input"
        >
          <b-form-input
            id="heyblob-input"
            v-model="newForm.keyblob"
            :state="newForm.keyblobState"
            required
            placeholder="bar=="
            ref="keyblob"
          ></b-form-input>
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
      newForm: {},
      zones: []
    }
  },

  mounted () {
    axios
      .get('/api/zones')
      .then(response => (this.zones = response.data))
  },

  methods: {
    attachZone () {
      axios
        .post('/api/zones', {
          domain: this.newForm.domain,
          server: this.newForm.server,
          keyname: this.newForm.keyname,
          keyblob: this.newForm.keyblob
        })
        .then(
          (response) => {
            if (this.zones == null) this.zones = []
            this.zones.push(response.data)
          },
          (error) => {
            alert('Unable to attach the given zone: ' + error.response.data.errmsg)
          }
        )
    },

    deleteZone (zone) {
      axios
        .delete('/api/zones/' + zone.domain)
        .then(response => (
          axios
            .get('/api/zones')
            .then(response => (this.zones = response.data))
        ))
    },

    show (zone) {
      this.$router.push('/zones/' + zone.domain)
    },

    modalShown () {
      this.$refs.domainname.focus()
    },

    resetModal () {
      this.newForm.domain = ''
      this.newForm.server = ''
      this.newForm.keyname = ''
      this.newForm.keyblob = ''
      this.newForm.domainNameState = null
      this.newForm.domainServerState = null
      this.newForm.keyNameState = null
      this.newForm.keyBlobState = null
    },

    checkFormValidity () {
      const valid = this.$refs.form.checkValidity()
      this.newForm.domainNameState = valid ? 'valid' : 'invalid'
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

      this.attachZone()

      this.$nextTick(() => {
        this.$refs.modal.hide()
      })
    }

  }
}
</script>
