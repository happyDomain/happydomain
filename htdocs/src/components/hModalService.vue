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
  <b-modal id="modal-addSvc" :size="step === 2 ? 'lg' : ''" scrollable @ok="handleModalSvcOk">
    <template v-slot:modal-title>
      Add a new service to <span class="text-monospace">{{ dn | fqdn(domain.domain) }}</span>
    </template>
    <template v-slot:modal-footer="{ ok, cancel }">
      <b-button v-if="update" :disabled="deleteServiceInProgress || !svcData || svcData._svctype === 'abstract.Origin'" variant="danger" @click="deleteService(svcData)">
        <b-spinner v-if="deleteServiceInProgress" label="Spinning" small />
        {{ $t('service.delete') }}
      </b-button>
      <b-button variant="secondary" @click="cancel()">
        {{ $t('common.cancel') }}
      </b-button>
      <b-button v-if="step === 2 && update" :disabled="addServiceInProgress" form="addSvcForm" type="submit" variant="success">
        <b-spinner v-if="addServiceInProgress" label="Spinning" small />
        {{ $t('service.update') }}
      </b-button>
      <b-button v-else-if="step === 2" form="addSvcForm" type="submit" variant="primary">
        {{ $t('service.add') }}
      </b-button>
      <b-button v-else form="addSvcForm" type="submit" variant="primary">
        {{ $t('common.continue') }}
      </b-button>
    </template>
    <form id="addSvcForm" @submit.stop.prevent="handleModalSvcOk">
      <p v-if="step === 0">
        Add a new subdomain under <span class="text-monospace">{{ domain.domain }}</span>:
        <b-input-group :append="'.' + domain.domain">
          <b-input v-model="dn" autofocus class="text-monospace" placeholder="new.subdomain" :state="newDomainState" @update="validateNewSubdomain" />
        </b-input-group>
      </p>
      <b-tabs v-else-if="step === 1" class="mb-2" content-class="mt-3">
        <b-tab title="Services" active>
          <p>
            Select a new service to add to <span class="text-monospace">{{ dn | fqdn(domain.domain) }}</span>:
          </p>
          <b-list-group v-if="step === 1">
            <b-list-group-item v-for="(svc, idx) in availableNewServices" :key="idx" :active="svcSelected === idx" button @click="svcSelected = idx">
              {{ svc.name }}
              <small class="text-muted">{{ svc.description }}</small>
              <b-badge v-for="(categorie, idcat) in svc.categories" :key="idcat" variant="gray" class="float-right ml-1">
                {{ categorie }}
              </b-badge>
            </b-list-group-item>
            <b-list-group-item v-for="(svc, idx) in disabledNewServices" :key="idx" :active="svcSelected === idx" disabled @click="svcSelected = idx">
              <span :title="svc.description">{{ svc.name }}</span> <small class="font-italic text-danger">{{ filteredNewServices[idx] }}</small>
              <b-badge v-for="(categorie, idcat) in svc.categories" :key="idcat" variant="gray" class="float-right ml-1">
                {{ categorie }}
              </b-badge>
            </b-list-group-item>
          </b-list-group>
        </b-tab>
        <b-tab title="Providers">
          <p>
            Select a new provider to add to <span class="text-monospace">{{ dn | fqdn(domain.domain) }}</span>:
          </p>
        </b-tab>
      </b-tabs>
      <div v-else-if="step === 2">
        <p>
          Fill the information for the {{ services[svcSelected].name }} at <span class="text-monospace">{{ dn | fqdn(domain.domain) }}</span>:
        </p>
        <h-resource-value ref="addModalResources" v-model="svcData.Service" edit :services="services" :type="svcSelected" />
      </div>
    </form>
  </b-modal>
</template>

<script>
import SourcesApi from '@/services/SourcesApi'
import ValidateDomain from '@/mixins/validateDomain'
import ZoneApi from '@/services/ZoneApi'

export default {
  name: 'HModalAddService',

  components: {
    hResourceValue: () => import('@/components/hResourceValue')
  },

  mixins: [ValidateDomain],

  props: {
    domain: {
      type: Object,
      required: true
    },
    myServices: {
      type: Object,
      required: true
    },
    services: {
      type: Object,
      required: true
    },
    zoneId: {
      type: Number,
      required: true
    }
  },

  data: function () {
    return {
      addServiceInProgress: false,
      availableResourceTypes: [],
      deleteServiceInProgress: false,
      dn: '',
      newDomainState: null,
      step: 0,
      svcData: {},
      svcSelected: null,
      update: false
    }
  },

  computed: {
    isLoading () {
      return this.availableResourceTypes.length > 0
    },

    availableNewServices () {
      var ret = {}

      for (const type in this.services) {
        if (this.filteredNewServices[type] == null) {
          ret[type] = this.services[type]
        }
      }

      return ret
    },

    disabledNewServices () {
      var ret = {}

      for (const type in this.services) {
        if (this.filteredNewServices[type] != null) {
          ret[type] = this.services[type]
        }
      }

      return ret
    },

    filteredNewServices () {
      return this.analyzeRestrictions(this.services)
    }
  },

  created () {
    SourcesApi.getAvailableResourceTypes(this.domain.id_source)
      .then(
        (response) => {
          this.availableResourceTypes = response.data
        }
      )
  },

  methods: {
    analyzeRestrictions (allServices) {
      var ret = {}

      for (const type in allServices) {
        const svc = allServices[type]

        if (svc.restrictions) {
          // Handle NeedTypes restriction: hosting provider need to support given types.
          if (svc.restrictions.needTypes) {
            for (const k in svc.restrictions.needTypes) {
              if (this.availableResourceTypes.indexOf(svc.restrictions.needTypes[k]) < 0) {
                ret[type] = 'is not available on this domain name hosting provider.'
                continue
              }
            }
          }

          // Handle Alone restriction: only nearAlone are allowed.
          if (svc.restrictions.alone && this.myServices.services[this.dn]) {
            var found = false
            for (const k in this.myServices.services[this.dn]) {
              const s = this.myServices.services[this.dn][k]
              if (s._svctype !== type && allServices[s._svctype].restrictions && !allServices[s._svctype].restrictions.nearAlone) {
                found = true
                break
              }
            }
            if (found) {
              ret[type] = 'has to be the only one in the subdomain.'
              continue
            }
          }

          // Handle Exclusive restriction: service can't be present along with another listed one.
          if (svc.restrictions.exclusive && this.myServices.services[this.dn]) {
            found = null
            for (const k in this.myServices.services[this.dn]) {
              const s = this.myServices.services[this.dn][k]
              for (const i in svc.restrictions.exclusive) {
                if (s._svctype === svc.restrictions.exclusive[i]) {
                  found = s._svctype
                  break
                }
              }
            }
            if (found) {
              ret[type] = 'cannot be present along with ' + allServices[found].name + '.'
              continue
            }
          }

          // Handle rootOnly restriction.
          if (svc.restrictions.rootOnly && this.dn !== '') {
            ret[type] = 'can only be present at the root of your domain.'
            continue
          }

          // Handle Single restriction: only one instance of the service per subdomain.
          if (svc.restrictions.single && this.myServices.services[this.dn]) {
            found = false
            for (const k in this.myServices.services[this.dn]) {
              const s = this.myServices.services[this.dn][k]
              if (s._svctype === type) {
                found = true
                break
              }
            }
            if (found) {
              ret[type] = 'can only be present once per subdomain.'
              continue
            }
          }

          // Handle presence of Alone and Leaf service in subdomain already.
          var oneAlone = null
          var oneLeaf = null
          for (const k in this.myServices.services[this.dn]) {
            const s = this.myServices.services[this.dn][k]
            if (this.services[s._svctype] && this.services[s._svctype].restrictions && this.services[s._svctype].restrictions.alone) {
              oneAlone = s._svctype
            }
            if (this.services[s._svctype] && this.services[s._svctype].restrictions && this.services[s._svctype].restrictions.leaf) {
              oneLeaf = s._svctype
            }
          }
          if (oneAlone && oneAlone !== type && !svc.restrictions.nearAlone) {
            ret[type] = 'cannot be present along with ' + allServices[oneAlone].name + ', that requires to be the only one in this subdomain.'
            continue
          }
          if (oneLeaf && oneLeaf !== type && !svc.restrictions.glue) {
            ret[type] = 'cannot be present along with ' + allServices[oneAlone].name + ', that requires to don\'t have subdomains.'
            continue
          }
        }
      }

      return ret
    },

    deleteService (service) {
      this.deleteServiceInProgress = true
      ZoneApi.deleteZoneService(this.domain.domain, this.zoneId, service)
        .then(
          (response) => {
            this.$bvModal.hide('modal-addSvc')
            this.deleteServiceInProgress = false
            this.$emit('updateMyServices', response.data)
          },
          (error) => {
            this.deleteServiceInProgress = false
            this.$bvToast.toast(
              error.response.data.errmsg, {
                title: 'An error occurs when deleting the service!',
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
          })
    },

    handleModalSvcOk (bvModalEvt) {
      bvModalEvt.preventDefault()

      if (this.step === 0 && this.dn !== '') {
        if (this.validateNewSubdomain()) {
          this.step = 1
        }
      } else if (this.step === 1 && this.svcSelected !== null) {
        this.step = 2
        this.svcData = { Service: {}, _svctype: this.svcSelected }
      } else if (this.step === 2 && this.svcSelected !== null) {
        this.$refs.addModalResources.saveChildrenValues()

        var func = null
        if (this.update) {
          func = ZoneApi.updateZoneService
        } else {
          func = ZoneApi.addZoneService
        }

        func(this.domain.domain, this.zoneId, this.dn, this.svcData)
          .then(
            (response) => {
              this.$emit('updateMyServices', response.data)
              this.$nextTick(() => {
                this.$bvModal.hide('modal-addSvc')
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
    },

    show (dn, data) {
      this.addServiceInProgress = false
      this.deleteServiceInProgress = false
      this.newDomainState = null

      if (dn !== undefined) {
        this.step = 1
        this.dn = dn
      } else {
        this.step = 0
        this.dn = ''
      }

      if (data !== undefined) {
        this.step = 2
        this.svcSelected = data._svctype
        this.svcData = data
        this.update = true
      } else {
        this.svcSelected = null
        this.svcData = { Service: {} }
        this.update = false
      }

      this.$bvModal.show('modal-addSvc')
    },

    validateNewSubdomain () {
      this.newDomainState = this.validateDomain(this.dn, true)
      return this.newDomainState
    }
  }
}
</script>
