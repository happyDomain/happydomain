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
      <strong>{{ $t('errors.error') }}</strong> {{ error }}
    </b-alert>
    <div v-if="!domain && error.length == 0" class="text-center">
      <b-spinner label="Spinning" />
      <p>{{ $t('wait.loading') }}</p>
    </div>
    <b-row style="min-height: inherit">
      <b-col sm="4" md="3" class="bg-light pb-5">
        <router-link to="/domains/" class="btn font-weight-bolder">
          <b-icon icon="chevron-up" />
        </router-link>
        <b-nav pills vertical variant="secondary">
          <b-nav-item :to="'/domains/' + domain.domain" :active="$route.name == 'domain-home'">
            {{ $t('domains.view.summary') }}
          </b-nav-item>
          <b-nav-item :to="'/domains/' + domain.domain + '/abstract'" :active="$route.name == 'domain-abstract'">
            {{ $t('domains.view.abstract') }}
          </b-nav-item>
          <b-nav-item :to="'/zones/' + domain.domain + '/records'" :active="$route.name == 'zone-records'">
            {{ $t('domains.view.live') }}
          </b-nav-item>
          <b-nav-item :to="'/domain/' + domain.domain + '/monitoring'" :active="$route.name == 'domain-monitoring'">
            {{ $t('domains.view.monitoring') }}
          </b-nav-item>
          <b-nav-item :to="'/domains/' + domain.domain + '/source'" :active="$route.name == 'domain-source'">
            {{ $t('domains.view.source') }}
          </b-nav-item>
          <hr>
          <b-nav-form>
            <b-button type="button" variant="outline-danger" @click="detachDomain()">
              <b-icon icon="trash-fill" /> {{ $t('domains.stop') }}
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
    this.$on('update-domain-info', this.updateDomainInfo)
  },

  methods: {
    detachDomain () {
      this.$bvModal.msgBoxConfirm(this.$t('domains.alert.remove', { domain: this.domain.domain }), {
        title: this.$t('domains.removal'),
        size: 'lg',
        okVariant: 'danger',
        okTitle: this.$t('domains.discard'),
        cancelVariant: 'outline-secondary',
        cancelTitle: this.$t('domains.view.cancel-title')
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
            this.$root.$bvToast.toast(this.$t('domains.alert.unable-retrieve.description', { domain: this.$route.params.domain }) + ' ' + error.response.data.errmsg, {
              title: this.$t('domains.alert.unable-retrieve.title'),
              autoHideDelay: 5000,
              variant: 'danger',
              toaster: 'b-toaster-content-right'
            })
            this.$router.push('/domains/')
          })
    }
  }
}
</script>
