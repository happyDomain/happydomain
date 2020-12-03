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
  <b-row v-if="!isLoading" class="pt-3">
    <b-col :sm="showSubdomainsList?9:null">
      <b-button v-show="!showSubdomainsList" class="float-right" variant="outline-secondary" @click="showSubdomainsList = !showSubdomainsList">
        <b-icon icon="list" aria-hidden="true" /><br>
      </b-button>

      <h-subdomain-item
        v-for="(dn, index) in sortedDomains"
        :key="index"
        :display-format="displayFormat"
        :dn="dn"
        :origin="domain.domain"
        :show-subdomains-list="showSubdomainsList"
        :services="services"
        :zone-services="myServices.services[dn]===undefined?[]:myServices.services[dn]"
        :aliases="aliases[dn]===undefined?[]:aliases[dn]"
        :zone-id="zoneId"
        @show-service-window="showServiceWindow"
        @update-my-services="updateMyServices"
        @add-subdomain="addSubdomain"
        @add-new-alias="addNewAlias"
        @add-new-service="addNewService"
      />

      <h-modal-service
        v-if="myServices"
        ref="modalService"
        :domain="domain"
        :my-services="myServices"
        :services="services"
        :zone-id="zoneId"
        @update-my-services="updateMyServices"
      />

      <b-modal id="modal-addAlias" title="Add a new alias" @ok="handleModalAliasSubmit">
        <template #modal-footer="{ cancel }">
          <b-button variant="secondary" @click="cancel()">
            {{ $t('common.cancel') }}
          </b-button>
          <b-button form="addAliasForm" type="submit" variant="primary">
            {{ $t('domains.add-alias') }}
          </b-button>
        </template>
        <form v-if="modal && modal.dn != null" id="addAliasForm" @submit.stop.prevent="handleModalAliasSubmit">
          <i18n path="domains.alias-creation">
            <span class="text-monospace">{{ modal.dn | fqdn(domain.domain) }}</span>
          </i18n>
          <b-input-group :append="'.' + domain.domain">
            <b-input v-model="modal.alias" autofocus class="text-monospace" placeholder="new.subdomain" :state="modal.newDomainState" @update="validateNewAlias" />
          </b-input-group>
          <div v-show="modal.alias" class="mt-3 text-center">
            {{ $t('domains.alias-creation-sample') }}
            <span class="text-monospace text-no-wrap">{{ modal.alias | fqdn(domain.domain) }}</span>
            <b-icon class="mr-1 ml-1" icon="arrow-right" />
            <span class="text-monospace text-no-wrap">{{ modal.dn | fqdn(domain.domain) }}</span>
          </div>
        </form>
      </b-modal>
    </b-col>
    <b-col v-show="showSubdomainsList" sm="3" class="sticky-top" style="margin-top: -10px; padding-top: 10px; height: 100vh; overflow-y: auto; z-index: 5">
      <b-button v-show="showSubdomainsList" class="float-right mb-2" variant="secondary" @click="showSubdomainsList = !showSubdomainsList">
        <b-icon icon="list" aria-hidden="true" /><br>
      </b-button>
      <b-button type="button" variant="outline-secondary" size="sm" class="ml-2 w-100" @click="addSubdomain()">
        <b-icon icon="server" />
        {{ $t('domains.add-a-subdomain') }}
      </b-button>
      <a
        v-for="(dn, index) in sortedDomains"
        :key="index"
        :href="'#' + (dn?dn:'@')"
        :title="dn | fqdn(domain.domain)"
        class="d-block text-truncate text-monospace text-muted"
        :style="'max-width: none; padding-left: ' + (dn === '' ? 0 : (dn.split('.').length * 10)) + 'px'"
      >
        {{ dn | fqdn(domain.domain) }}
      </a>
    </b-col>
  </b-row>
</template>

<script>
import ServiceSpecsApi from '@/services/ServiceSpecsApi'
import ValidateDomain from '@/mixins/validateDomain'
import ZoneApi from '@/services/ZoneApi'
import { domainCompare } from '@/utils/domainCompare'

export default {
  name: 'HSubdomainList',

  components: {
    hModalService: () => import('@/components/hModalService'),
    hSubdomainItem: () => import('@/components/hSubdomainItem')
  },

  mixins: [ValidateDomain],

  props: {
    domain: {
      type: Object,
      required: true
    },
    displayFormat: {
      type: String,
      default: 'grid'
    },
    zoneId: {
      type: Number,
      required: true
    }
  },

  data: function () {
    return {
      hideDomain: {},
      modal: null,
      myServices: null,
      services: {},
      showSubdomainsList: false,
      updateServiceInProgress: false
    }
  },

  computed: {
    aliases () {
      var ret = {}

      for (const dn in this.myServices.services) {
        this.myServices.services[dn].forEach(function (svc) {
          if (svc._svctype === 'svcs.CNAME') {
            if (!ret[svc.Service.Target]) {
              ret[svc.Service.Target] = []
            }
            ret[svc.Service.Target].push(dn)
          }
        })
      }

      return ret
    },

    isLoading () {
      return this.myServices == null && this.zoneId === undefined && this.services === {}
    },

    sortedDomains () {
      if (this.myServices == null) {
        return []
      }

      var domains = Object.keys(this.myServices.services)
      domains.sort(domainCompare)

      return domains
    }
  },

  watch: {
    domain: function () {
      this.pullZone()
    },
    zoneId: function () {
      this.pullZone()
    }
  },

  created () {
    this.pullZone()

    ServiceSpecsApi.getServiceSpecs()
      .then(
        (response) => (this.services = response.data)
      )
  },

  methods: {
    addNewService (subdomain) {
      this.$refs.modalService.show(subdomain)
    },

    addNewAlias (subdomain) {
      this.modal = {
        dn: subdomain,
        alias: ''
      }
      this.$bvModal.show('modal-addAlias')
    },

    addSubdomain () {
      this.$refs.modalService.show()
    },

    showServiceWindow (service) {
      this.$refs.modalService.show(service._domain, service)
    },

    fakeSaveService (cbSuccess) {
      if (cbSuccess) {
        cbSuccess()
      }
    },

    goToAnchor () {
      var hash = this.$route.hash.substr(1)
      if (!this.isLoading && hash.length > 0) {
        setTimeout(function () {
          window.scrollTo(0, document.getElementById(hash).offsetTop)
        }, 500)
      }
    },

    handleModalAliasSubmit (bvModalEvt) {
      bvModalEvt.preventDefault()

      if (this.modal.alias) {
        if (this.validateNewAlias()) {
          ZoneApi
            .addZoneService(this.domain.domain, this.zoneId, this.modal.alias, { Service: { target: this.modal.dn || '@' }, _svctype: 'svcs.CNAME' })
            .then(
              (response) => {
                this.myServices = response.data
                this.$nextTick(() => {
                  this.$bvModal.hide('modal-addAlias')
                })
              },
              (error) => {
                this.$root.$bvToast.toast(
                  error.response.data.errmsg, {
                    title: 'Unable to add the new service',
                    autoHideDelay: 5000,
                    variant: 'danger',
                    toaster: 'b-toaster-content-right'
                  }
                )
              }
            )
        }
      }
    },

    pullZone () {
      if (this.domain === undefined || this.domain.domain === undefined || this.zoneId === undefined) {
        return
      }

      ZoneApi
        .getZone(this.domain.domain, this.zoneId)
        .then(
          (response) => {
            this.myServices = response.data
            // this.goToAnchor()
          },
          (error) => {
            this.$root.$bvToast.toast(
              'Unfortunately, we were unable to retrieve information for the domain ' + this.domain.domain + ': ' + error.response.data.errmsg, {
                title: 'Unable to retrieve domain information',
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
            this.$router.push('/domains/' + encodeURIComponent(this.domain.domain))
          }
        )
    },

    updateMyServices (myS) {
      this.myServices = myS
    },

    validateNewAlias () {
      if (this.myServices.services) {
        for (const dn in this.myServices.services) {
          if (this.modal.alias === dn) {
            this.modal.newDomainState = false
            return false
          }
        }
      }

      this.modal.newDomainState = this.validateDomain(this.modal.alias, true)

      return this.modal.newDomainState
    }
  }
}
</script>
