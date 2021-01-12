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
  <b-container class="d-flex flex-column" :fluid="responses?true:false">
    <b-row class="flex-grow-1">
      <b-col :offset-md="responses?0:2" :md="responses?4:8" :class="(responses?'bg-light ':'') + 'pt-4 pb-5 sticky-top'">
        <h1 class="text-center mb-3">
          {{ $t('menu.dns-resolver') }}
        </h1>
        <form class="pt-3 pb-5" @submit.stop.prevent="submitRequest">
          <b-form-group
            id="input-domain"
            :label="$t('common.domain')"
            label-for="domain"
          >
            <template slot="description">
              <i18n path="resolver.domain-description">
                <router-link to="/resolver/wikipedia.org" class="text-monospaced">
                  wikipedia.org
                </router-link>
              </i18n>
            </template>
            <b-form-input
              id="domain"
              v-model="form.domain"
              required
              placeholder="happydns.org"
            />
          </b-form-group>

          <div class="text-center mb-3">
            <b-button v-b-toggle.resolver-advanced-settings variant="secondary">
              {{ $t('resolver.advanced') }}
            </b-button>
          </div>

          <b-collapse id="resolver-advanced-settings">
            <b-form-group
              id="input-type"
              :label="$t('common.field')"
              label-for="type"
            >
              <template slot="description">
                <i18n path="resolver.field-description">
                  <a href="//help.happydns.org/tools/resolver" target="_blank">{{ $t('resolver.field-description-more-info') }}</a>
                </i18n>
              </template>
              <b-form-select
                id="type"
                v-model="form.type"
                required
                :options="existing_types"
              />
            </b-form-group>

            <b-form-group
              id="input-resolver"
              :label="$t('common.resolver')"
              label-for="resolver"
              :description="$t('resolver.resolver-description')"
            >
              <b-form-select
                id="resolver"
                v-model="form.resolver"
                required
              >
                <b-form-select-option-group
                  v-for="(group, gname) in existing_resolvers"
                  :key="gname"
                  :label="gname"
                  :options="group"
                />
                <b-form-select-option value="custom">
                  {{ $t('resolver.custom') }}
                </b-form-select-option>
              </b-form-select>
            </b-form-group>

            <b-form-group
              v-show="form.resolver === 'custom'"
              id="input-custom-resolver"
              :label="$t('resolver.custom')"
              label-for="custom-resolver"
              :description="$t('resolver.custom-description')"
            >
              <b-form-input
                id="custom-resolver"
                v-model="form.custom"
                :required="form.resolver === 'custom'"
                placeholder="127.0.0.1"
              />
            </b-form-group>

            <b-form-checkbox
              id="showDNSSEC"
              v-model="showDNSSEC"
              name="showDNSSEC"
              class="mb-3"
            >
              {{ $t('resolver.showDNSSEC') }}
            </b-form-checkbox>
          </b-collapse>

          <div class="ml-3 mr-3">
            <b-button class="float-right" type="submit" variant="primary" :disabled="request_pending">
              <b-spinner v-if="request_pending" :label="$t('common.spinning')" small />
              {{ $t('common.run') }}
            </b-button>
          </div>
        </form>
      </b-col>
      <b-col v-if="responses === 'no-answer'" md="8" class="pt-2">
        <h3>{{ $tc('common.records', 0, { type: form.type }) }}</h3>
      </b-col>
      <b-col v-else-if="responses" md="8" class="pt-2">
        <div v-for="(rrs,type) in responseByType" :key="type">
          <h3>{{ $tc('common.records', rrs.length, { type: $options.filters.nsrrtype(type) }) }}</h3>
          <table class="table table-hover table-sm">
            <thead>
              <h-record-head :rrtype="type" />
            </thead>
            <tbody>
              <h-record v-for="(rr,index) in rrs" :key="index" :record="rr" />
            </tbody>
          </table>
        </div>
      </b-col>
    </b-row>
  </b-container>
</template>

<script>
import axios from 'axios'

export default {

  components: {
    hRecord: () => import('@/components/hRecord'),
    hRecordHead: () => import('@/components/hRecordHead')
  },

  data: function () {
    return {
      request_pending: false,
      existing_types: ['ANY', 'A', 'AAAA', 'NS', 'SRV', 'MX', 'TXT', 'SOA'],
      existing_resolvers: {
        Unfiltered:
        [
          { value: 'local', text: 'Local resolver' },
          { value: '1.1.1.1', text: 'Cloudflare DNS resolver' },
          { value: '4.2.2.1', text: 'Level3 resolver' },
          { value: '8.8.8.8', text: 'Google Public DNS resolver' },
          { value: '9.9.9.10', text: 'Quad9 DNS resolver without security blocklist' },
          { value: '64.6.64.6', text: 'Verisign DNS resolver' },
          { value: '74.82.42.42', text: 'Hurricane Electric DNS resolver' },
          { value: '208.67.222.222', text: 'OpenDNS resolver' },
          { value: '8.26.56.26', text: 'Comodo Secure DNS resolver' },
          { value: '199.85.126.10', text: 'Norton ConnectSafe DNS resolver' },
          { value: '198.54.117.10', text: 'SafeServe DNS resolver' },
          { value: '84.200.69.80', text: 'DNS.WATCH resolver' },
          { value: '185.121.177.177', text: 'OpenNIC DNS resolver' },
          { value: '37.235.1.174', text: 'FreeDNS resolver' },
          { value: '80.80.80.80', text: 'Freenom World DNS resolver' },
          { value: '216.131.65.63', text: 'StrongDNS resolver' },
          { value: '94.140.14.140', text: 'AdGuard non-filtering DNS resolver' },
          { value: '91.239.100.100', text: 'Uncensored DNS resolver' },
          { value: '216.146.35.35', text: 'Dyn DNS resolver' },
          { value: '77.88.8.8', text: 'Yandex.DNS resolver' },
          { value: '129.250.35.250', text: 'NTT DNS resolver' },
          { value: '223.5.5.5', text: 'AliDNS resolver' },
          { value: '1.2.4.8', text: 'CNNIC SDNS resolver' },
          { value: '119.29.29.29', text: 'DNSPod resolver' },
          { value: '114.215.126.16', text: 'oneDNS resolver' },
          { value: '124.251.124.251', text: 'cloudxns resolver' },
          { value: '114.114.114.114', text: 'Baidu DNS resolver' },
          { value: '156.154.70.1', text: 'DNS Advantage resolver' },
          { value: '87.118.111.215', text: 'FoolDNS resolver' },
          { value: '101.101.101.101', text: 'Quad 101 DNS resolver' },
          { value: '114.114.114.114', text: '114DNS resolver' },
          { value: '168.95.1.1', text: 'HiNet DNS resolver' },
          { value: '80.67.169.12', text: 'French Data Network DNS resolver' },
          { value: '81.218.119.11', text: 'GreenTeamDNS resolver' },
          { value: '208.76.50.50', text: 'SmartViper DNS resolver' },
          { value: '23.253.163.53', text: 'Alternate DNS resolver' },
          { value: '109.69.8.51', text: 'puntCAT DNS resolver' },
          { value: '156.154.70.1', text: 'Neustar DNS resolver' },
          { value: '101.226.4.6', text: 'DNSpai resolver' }
          // Your open resolver here? Don't hesitate to contribute to the project!
        ],
        Filtered: [
          { value: '1.1.1.2', text: 'Cloudflare Malware Blocking Only DNS resolver' },
          { value: '1.1.1.3', text: 'Cloudflare Malware and Adult Content Blocking Only DNS resolver' },
          { value: '9.9.9.9', text: 'Quad9 DNS resolver' },
          { value: '94.140.14.14', text: 'AdGuard default DNS resolver' },
          { value: '94.140.14.15', text: 'AdGuard family protection DNS resolver' },
          { value: '77.88.8.2', text: 'Yandex.DNS Safe resolver' },
          { value: '77.88.8.3', text: 'Yandex.DNS Family resolver' },
          { value: '156.154.70.2', text: 'DNS Advantage Threat Protection resolver' },
          { value: '156.154.70.3', text: 'DNS Advantage Family Secure resolver' },
          { value: '156.154.70.4', text: 'DNS Advantage Business Secure resolver' },
          { value: '185.228.168.168', text: 'CleanBrowsing Family Filter DNS resolver' },
          { value: '185.228.168.10', text: 'CleanBrowsing Adult Filter DNS resolver' }
          // Your open resolver here? Don't hesitate to contribute to the project!
        ]
      },
      form: {
        resolver: 'local',
        type: 'ANY'
      },
      responses: null,
      showDNSSEC: false
    }
  },

  computed: {
    responseByType () {
      const ret = {}

      for (const i in this.filteredResponses) {
        if (!ret[this.filteredResponses[i].Hdr.Rrtype]) {
          ret[this.filteredResponses[i].Hdr.Rrtype] = []
        }
        ret[this.filteredResponses[i].Hdr.Rrtype].push(this.filteredResponses[i])
      }

      return ret
    },

    filteredResponses () {
      if (!this.responses) {
        return []
      }

      if (this.showDNSSEC) {
        return this.responses
      } else {
        return this.responses.filter(rr => (rr.Hdr.Rrtype !== 46 && rr.Hdr.Rrtype !== 47 && rr.Hdr.Rrtype !== 50))
      }
    }
  },

  watch: {
    $route (n) {
      if (n.params.domain && (!this.response || n.params.domain !== this.form.domain)) {
        this.form.domain = n.params.domain
        this.submitRequest()
      } else if (!n.params.domain && this.responses) {
        this.responses = null
      }
    }
  },

  mounted () {
    if (this.$route.params.domain) {
      this.form.domain = this.$route.params.domain
      this.submitRequest()
    }
  },

  methods: {
    submitRequest () {
      this.request_pending = true
      axios
        .post('/api/resolver', this.form)
        .then(
          (response) => {
            if (response.data.Answer) {
              this.responses = response.data.Answer
            } else {
              this.responses = 'no-answer'
            }
            this.request_pending = false
          },
          (error) => {
            this.$bvToast.toast(
              error.response.data.errmsg, {
                title: this.$t('errors.resolve'),
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
            this.request_pending = false
          })
      if (this.$route.params.domain !== this.form.domain) {
        this.$router.push('/resolver/' + encodeURIComponent(this.form.domain))
      }
    }
  }
}
</script>

<style>
  .form-group label {
    font-weight: bold;
  }
</style>
