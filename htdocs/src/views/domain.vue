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
  <b-container fluid>
    <b-alert variant="danger" :show="error.length != 0">
      <strong>Error:</strong> {{ error }}
    </b-alert>
    <div v-if="!domain && error.length == 0" class="text-center">
      <b-spinner label="Spinning" />
      <p>Loading the domain&nbsp;&hellip;</p>
    </div>
    <b-row>
      <b-col sm="4" md="3" class="bg-light pb-5">
        <router-link to="/domains/" class="btn font-weight-bolder">
          <b-icon icon="chevron-up" />
        </router-link>
        <b-nav pills vertical variant="secondary">
          <b-nav-item :to="'/domains/' + domain.domain" :active="$route.name == 'domain-home'">
            Summary
          </b-nav-item>
          <b-nav-item :to="'/domains/' + domain.domain + '/abstract'" :active="$route.name == 'domain-abstract'">
            Abstract zone
          </b-nav-item>
          <b-nav-item :to="'/zones/' + domain.domain + '/records'" :active="$route.name == 'zone-records'">
            View records
          </b-nav-item>
          <b-nav-item :to="'/domain/' + domain.domain + '/monitoring'" :active="$route.name == 'domain-monitoring'">
            Monitoring
          </b-nav-item>
          <b-nav-item :to="'/domains/' + domain.domain + '/source'" :active="$route.name == 'domain-source'">
            Domain source
          </b-nav-item>
          <hr>
          <b-nav-form>
            <b-button type="button" variant="outline-danger" @click="detachDomain()">
              <b-icon icon="trash-fill" /> Stop managing this domain
            </b-button>
          </b-nav-form>
        </b-nav>
      </b-col>
      <b-col sm="8" md="9" class="mb-5">
        <router-view :domain="domain" />
      </b-col>
    </b-row>
  </b-container>
</template>

<script>
import axios from 'axios'

export default {

  data: function () {
    return {
      error: '',
      domain: {}
    }
  },

  mounted () {
    this.updateDomainInfo()
    this.$on('updateDomainInfo', this.updateDomainInfo)
  },

  methods: {
    detachDomain () {
      this.$bvModal.msgBoxConfirm('This action will permanently remove the domain ' + this.domain.domain + ' from your managed domains. All history and abstracted zones will be discarded. This action will not delete or unregister your domain from your provider, nor alterate what is currently served. It will only affect what you see in happyDNS. Are you sure you want to continue?', {
        title: 'Confirm Domain Removal',
        size: 'lg',
        okVariant: 'danger',
        okTitle: 'Discard',
        cancelVariant: 'outline-secondary',
        cancelTitle: 'Keep my domain in happyDNS'
      })
        .then(value => {
          if (value) {
            axios
              .delete('/api/domains/' + encodeURIComponent(this.domain.domain))
              .then(response => (
                this.$router.push('/domains/')
              ))
          }
        })
    },

    updateDomainInfo () {
      var mydomain = this.$route.params.domain
      axios
        .get('/api/domains/' + encodeURIComponent(mydomain))
        .then(
          response => (this.domain = response.data),
          error => {
            this.$root.$bvToast.toast(
              'Unfortunately, we were unable to retrieve information for the domain ' + this.$route.params.domain + ': ' + error.response.data.errmsg, {
                title: 'Unable to retrieve domain information',
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
            this.$router.push('/domains/')
          })
    }
  }
}
</script>
