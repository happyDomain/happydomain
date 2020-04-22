<template>
  <b-container class="mt-4">
    <h1 class="text-center mb-4">Welcome to <span style="font-family: 'Fortheenas01';font-weight:bold;">happy<span style="font-family: 'Fortheenas01 Bold';margin-left:.1em;">DNS</span></span>!</h1>
    <b-row>
    <div class="offset-md-2 col-md-8">
    <b-list-group>
      <b-list-group-item v-if="loading" class="text-center">
        <b-spinner variant="secondary" label="Spinning"></b-spinner> Retrieving your domains...
      </b-list-group-item>
      <b-list-group-item :to="'domains/' + domain.domain" v-for="(domain, index) in domains" v-bind:key="index" class="d-flex justify-content-between align-items-center">
        {{ domain.domain }}
        <b-badge variant="success">OK</b-badge>
      </b-list-group-item>
    </b-list-group>
    <b-list-group class="mt-2">
      <form @submit.stop.prevent="showModal" v-if="!loading">
        <b-list-group-item class="d-flex justify-content-between align-items-center">
          <b-input-group size="sm">
            <b-input-group-prepend>
              <b-icon icon="plus"></b-icon>
            </b-input-group-prepend>
            <input placeholder="my.new.domain" v-model="newForm.domain" style="border:none; flex: 1 1 auto;">
            <b-input-group-append v-show="newForm.domain.length">
              <b-button type="submit" variant="outline-primary">Add new domain</b-button>
            </b-input-group-append>
          </b-input-group>
        </b-list-group-item>
      </form>
    </b-list-group>
      </div>
      </b-row>

    <b-modal
      id="newDomainModal"
      ref="modal"
      title="Attach new domain"
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
          <b-form-group label="Storage facility">
            <b-form-radio v-model="newForm.storage_facility" name="storage-facility" value="live">Live only</b-form-radio>
            <b-form-radio v-model="newForm.storage_facility" name="storage-facility" value="historical">With history</b-form-radio>
          </b-form-group>
        </b-form-group>
      </form>
    </b-modal>
  </b-container>
</template>

<script>
import axios from 'axios'

export default {

  data: function () {
    return {
      domainNameState: null,
      loading: true,
      newForm: { domain: '', storage_facility: 'live' },
      domains: []
    }
  },

  mounted () {
    setTimeout(() =>
      axios
        .get('/api/domains')
        .then(response => { this.domains = response.data; this.loading = false; return true })
    , 100)
  },

  methods: {
    show (domain) {
      this.$router.push('/domains/' + domain.domain)
    },

    showModal () {
      this.$bvModal.show('newDomainModal')
    },

    modalShown () {
      this.$refs.domainserver.focus()
    },

    resetModal () {
      this.newForm.server = ''
      this.newForm.keyname = ''
      this.newForm.keyblob = ''
      this.newForm.storage_facility = 'live'
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

    attachDomain () {
      axios
        .post('/api/domains', {
          domain: this.newForm.domain,
          server: this.newForm.server,
          keyname: this.newForm.keyname,
          keyblob: this.newForm.keyblob,
          storage_facility: this.newForm.storage_facility
        })
        .then(
          (response) => {
            if (this.domains == null) this.domains = []
            this.domains.push(response.data)
          },
          (error) => {
            alert('Unable to attach the given domain: ' + error.response.data.errmsg)
          }
        )
    },

    handleSubmit () {
      if (!this.checkFormValidity()) {
        return
      }

      this.attachDomain()

      this.$nextTick(() => {
        this.$refs.modal.hide()
      })
    }
  }
}
</script>
