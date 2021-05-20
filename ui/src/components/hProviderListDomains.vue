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
  <h-zone-list :domains="listImportableDomains" loading-str="wait.asking-domains" :parent-is-loading="isLoading">
    <template #badges="{ domain }">
      <b-badge v-if="domain.state" class="ml-1" :variant="domain.state">
        <b-icon v-if="domain.state === 'success'" icon="check" />
        <b-icon v-else-if="domain.state === 'info'" icon="exclamation-circle" />
        <b-icon v-else-if="domain.state === 'warning'" icon="exclamation-triangle" />
        <b-icon v-else-if="domain.state === 'danger'" icon="exclamation-octagon" />
        {{ domain.message }}
      </b-badge>
      <b-badge v-else-if="haveDomain(domain)" class="ml-1" variant="success">
        <b-icon icon="check" /> {{ $t('service.already') }}
      </b-badge>
      <b-button v-else type="button" class="ml-1" variant="primary" size="sm" @click="importDomain(domain)">
        {{ $t('domains.add-now') }}
      </b-button>
      <b-button v-if="domain.btn" type="button" class="ml-1" :variant="domain.state" size="sm" @click="doDomainAction(domain)">
        {{ $t(domain.btn) }}
      </b-button>
    </template>
    <template v-if="noDomainsList" #no-domain>
      <b-list-group-item class="text-center">
        {{ $t('errors.domain-list') }}
      </b-list-group-item>
    </template>
    <template v-else-if="!importableDomains || importableDomains.length === 0" #no-domain>
      <b-list-group-item class="text-center">
        {{ $t('errors.domain-have') }}
      </b-list-group-item>
    </template>
    <template v-else-if="listImportableDomains.length === 0" #no-domain>
      <b-list-group-item class="text-center">
        <i18n path="errors.domain-all-imported">
          {{ sourceSpecs_getAll[provider._srctype].name }}
        </i18n>
      </b-list-group-item>
    </template>
  </h-zone-list>
</template>

<script>
import { mapGetters } from 'vuex'
import ProvidersApi from '@/api/providers'
import AddDomainToProvider from '@/mixins/addDomainToProvider'

export default {

  components: {
    hZoneList: () => import('@/components/ZoneList')
  },

  mixins: [AddDomainToProvider],

  props: {
    showAlreadyImported: {
      type: Boolean,
      default: false
    },
    showDomainsWithActions: {
      type: Boolean,
      default: false
    },
    provider: {
      type: Object,
      required: true
    }
  },

  data: function () {
    return {
      domainsWithActions: null,
      importableDomains: null
    }
  },

  computed: {
    isLoading () {
      return this.importableDomains == null && !this.noDomainsList
    },

    listImportableDomains () {
      if (!this.importableDomains) {
        return []
      }

      let ret = this.importableDomains

      if (!this.showAlreadyImported) {
        ret = ret.filter(
          d => !this.domains_getAll.find(i => i.domain === d)
        )
      }

      ret = ret.map(d => ({ domain: d, id_provider: this.provider._id }))

      if (this.domainsWithActions && this.showDomainsWithActions) {
        this.domainsWithActions.forEach((dwa) => {
          ret.push({ domain: dwa.fqdn, state: dwa.state, message: dwa.message, btn: dwa.btn, action: dwa.action, id_provider: this.provider._id })
        })
      }

      return ret
    },

    noDomainsList () {
      return !this.sourceSpecs_getAll[this.provider._srctype] || !this.sourceSpecs_getAll[this.provider._srctype].capabilities || this.sourceSpecs_getAll[this.provider._srctype].capabilities.indexOf('ListDomains') === -1
    },

    ...mapGetters('sourceSpecs', ['sourceSpecs_getAll']),
    ...mapGetters('domains', ['domains_getAll'])
  },

  watch: {
    noDomainsList: function (ndl) {
      this.$emit('no-domains-list-change', ndl)
    },

    provider: function () {
      this.getImportableDomains()
    },

    sourceSpecs_getAll: function (ss) {
      if (ss) {
        this.getImportableDomains()
      }
    }
  },

  mounted () {
    if (this.provider) {
      this.getImportableDomains()
    }
    this.$emit('no-domains-list-change', this.noDomainsList)
  },

  methods: {
    doDomainAction (domain) {
      ProvidersApi.doProviderDomainsAction(this.provider._id, domain.domain, domain.action)
        .then(
          response => {
            this.$store.dispatch('domains/getAllMyDomains')
            if (response.data.errmsg) {
              this.$root.$bvToast.toast(
                response.data.errmsg, {
                  title: this.$t('domains.attached-new'),
                  autoHideDelay: 5000,
                  variant: 'success',
                  toaster: 'b-toaster-content-right'
                }
              )
            }
          },
          error => {
            this.$root.$bvToast.toast(
              error.response.data.errmsg, {
                title: this.$t('errors.domain-attach'),
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
          })
    },

    haveDomain (domain) {
      return this.domains_getAll.find(i => i.domain === domain.domain)
    },

    importAllDomains () {
      this.listImportableDomains.forEach((domain) => {
        if (!domain.state && !this.haveDomain(domain)) {
          this.importDomain(domain)
        }
      })
    },

    importDomain (domain) {
      const vm = this
      this.addDomainToProvider(this.provider, domain.domain, null, function (data) {
        vm.$store.dispatch('domains/getAllMyDomains')
      })
    },

    getImportableDomains () {
      this.importableDomains = null
      if (this.noDomainsList) {
        return
      }

      ProvidersApi.listProviderDomains(this.provider._id)
        .then(
          response => {
            if (response.data === null) {
              this.importableDomains = []
            } else {
              this.importableDomains = response.data
            }
          },
          error => {
            this.$root.$bvToast.toast(
              error.response.data.errmsg, {
                title: this.$t('errors.domain-access'),
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
            this.$router.replace('/providers/' + encodeURIComponent(this.myProvider._id))
          })

      if (!this.showDomainsWithActions || this.sourceSpecs_getAll[this.provider._srctype].capabilities.indexOf('ListDomainsWithActions') === -1) {
        return
      }

      ProvidersApi.listProviderDomainsWithActions(this.provider._id)
        .then(
          response => (this.domainsWithActions = response.data),
          error => {
            this.$root.$bvToast.toast(
              error.response.data.errmsg, {
                title: this.$t('errors.domain-access'),
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
          })
    }
  }
}
</script>
