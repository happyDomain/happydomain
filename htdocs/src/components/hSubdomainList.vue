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
  <div v-if="!isLoading" class="pt-3">
    <h-subdomain-item v-for="(dn, index) in sortedDomains" :key="index" :display-card="displayCard" :dn="dn" :origin="domain.domain" :services="services" :zone-services="myServices.services[dn]===undefined?[]:myServices.services[dn]" :aliases="aliases[dn]===undefined?[]:aliases[dn]" :zone-id="zoneId" @showServiceWindow="showServiceWindow($event)" @updateMyServices="updateMyServices($event)" @addSubdomain="addSubdomain()" @addNewAlias="addNewAlias($event)" @addNewService="addNewService($event)" />

    <b-modal id="modal-addSvc" :size="modal && modal.step === 2 ? 'lg' : ''" scrollable @ok="handleModalSvcOk">
      <template v-slot:modal-title>
        Add a new service to <span class="text-monospace">{{ modal.dn | fqdn(domain.domain) }}</span>
      </template>
      <form v-if="modal" @submit.stop.prevent="handleModalSvcOk">
        <p v-if="modal.step === 0">
          Add a new subdomain under <span class="text-monospace">{{ domain.domain }}</span>:
          <b-input-group :append="'.' + domain.domain">
            <b-input v-model="modal.dn" autofocus class="text-monospace" placeholder="new.subdomain" :state="modal.newDomainState" @update="validateNewSubdomain" />
          </b-input-group>
        </p>
        <p v-else-if="modal.step === 1">
          Select a new service to add to <span class="text-monospace">{{ modal.dn | fqdn(domain.domain) }}</span>:
        </p>
        <p v-else-if="modal.step === 2">
          Fill the information for the {{ services[modal.svcSelected].name }} at <span class="text-monospace">{{ modal.dn | fqdn(domain.domain) }}</span>:
        </p>
        <b-list-group v-if="modal.step === 1" class="mb-2">
          <b-list-group-item v-for="(svc, idx) in availableNewServices" :key="idx" :active="modal.svcSelected === idx" button @click="modal.svcSelected = idx">
            {{ svc.name }}
            <small class="text-muted">{{ svc.description }}</small>
            <b-badge v-for="(categorie, idcat) in svc.categories" :key="idcat" variant="gray" class="float-right ml-1">
              {{ categorie }}
            </b-badge>
          </b-list-group-item>
          <b-list-group-item v-for="(svc, idx) in disabledNewServices" :key="idx" :active="modal.svcSelected === idx" disabled @click="modal.svcSelected = idx">
            <span :title="svc.description">{{ svc.name }}</span> <small class="font-italic text-danger">{{ filteredNewServices[idx] }}</small>
            <b-badge v-for="(categorie, idcat) in svc.categories" :key="idcat" variant="gray" class="float-right ml-1">
              {{ categorie }}
            </b-badge>
          </b-list-group-item>
        </b-list-group>
        <h-resource-value v-else-if="modal.step === 2" v-model="modal.svcData" edit :services="services" :type="modal.svcSelected" @input="modal.svcData = $event" />
      </form>
    </b-modal>

    <b-modal id="modal-updSvc" size="xl" scrollable @ok="handleUpdateSvc">
      <template v-slot:modal-title>
        Update <span v-if="modal.svcData._svctype" :title="services[modal.svcData._svctype].description">{{ services[modal.svcData._svctype].name }} </span>on <span class="text-monospace">{{ modal.dn | fqdn(domain.domain) }}</span>
      </template>
      <form v-if="modal" @submit.stop.prevent="handleUpdateSvc">
        <h-resource-value v-model="modal.svcData.Service" edit :services="services" :type="modal.svcData._svctype" @input="modal.svcData.Service = $event" />
      </form>
    </b-modal>

    <b-modal id="modal-addAlias" title="Add a new alias" @ok="handleModalAliasSubmit">
      <form v-if="modal && modal.dn != null" @submit.stop.prevent="handleModalAliasSubmit">
        Add an alias pointing to <span class="text-monospace">{{ modal.dn | fqdn(domain.domain) }}</span>:
        <b-input-group :append="'.' + domain.domain">
          <b-input v-model="modal.alias" autofocus class="text-monospace" placeholder="new.subdomain" :state="modal.newDomainState" @update="validateNewAlias" />
        </b-input-group>
      </form>
    </b-modal>
  </div>
</template>

<script>
import ServiceSpecsApi from '@/services/ServiceSpecsApi'
import ZoneApi from '@/services/ZoneApi'

export default {
  name: 'HSubdomainList',

  components: {
    hSubdomainItem: () => import('@/components/hSubdomainItem'),
    hResourceValue: () => import('@/components/hResourceValue')
  },

  props: {
    domain: {
      type: Object,
      required: true
    },
    displayCard: {
      type: Boolean,
      default: false
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
      services: null
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
      var ret = {}

      for (const type in this.services) {
        const svc = this.services[type]

        if (svc.restrictions) {
          // Handle Alone restriction: only nearAlone are allowed.
          if (svc.restrictions.alone && this.myServices.services[this.modal.dn]) {
            var found = false
            for (const k in this.myServices.services[this.modal.dn]) {
              const s = this.myServices.services[this.modal.dn][k]
              if (s._svctype !== type && this.services[s._svctype].restrictions && !this.services[s._svctype].restrictions.nearAlone) {
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
          if (svc.restrictions.exclusive && this.myServices.services[this.modal.dn]) {
            found = null
            for (const k in this.myServices.services[this.modal.dn]) {
              const s = this.myServices.services[this.modal.dn][k]
              for (const i in svc.restrictions.exclusive) {
                if (s._svctype === svc.restrictions.exclusive[i]) {
                  found = s._svctype
                  break
                }
              }
            }
            if (found) {
              ret[type] = 'cannot be present along with ' + this.services[found].name + '.'
              continue
            }
          }

          // Handle rootOnly restriction.
          if (svc.restrictions.rootOnly && this.modal.dn !== '') {
            ret[type] = 'can only be present at the root of your domain.'
            continue
          }

          // Handle Single restriction: only one instance of the service per subdomain.
          if (svc.restrictions.single && this.myServices.services[this.modal.dn]) {
            found = false
            for (const k in this.myServices.services[this.modal.dn]) {
              const s = this.myServices.services[this.modal.dn][k]
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
          for (const k in this.myServices.services[this.modal.dn]) {
            const s = this.myServices.services[this.modal.dn][k]
            if (this.services[s._svctype].restrictions && this.services[s._svctype].restrictions.alone) {
              oneAlone = s._svctype
            }
            if (this.services[s._svctype].restrictions && this.services[s._svctype].restrictions.leaf) {
              oneLeaf = s._svctype
            }
          }
          if (oneAlone && oneAlone !== type && !svc.restrictions.nearAlone) {
            ret[type] = 'cannot be present along with ' + this.services[oneAlone].name + ', that requires to be the only one in this subdomain.'
            continue
          }
          if (oneLeaf && oneLeaf !== type && !svc.restrictions.glue) {
            ret[type] = 'cannot be present along with ' + this.services[oneAlone].name + ', that requires to don\'t have subdomains.'
            continue
          }
        }
      }

      return ret
    },

    isLoading () {
      return this.myServices == null && this.zoneId === undefined && this.services == null
    },

    sortedDomains () {
      if (this.myServices == null) {
        return []
      }

      var domains = Object.keys(this.myServices.services)
      domains.sort(function (a, b) {
        var as = a.split('.').reverse()
        var bs = b.split('.').reverse()

        var maxDepth = Math.min(as.length, bs.length)
        for (var i = 0; i < maxDepth; i++) {
          var cmp = as[i].localeCompare(bs[i])
          if (cmp !== 0) {
            return cmp
          }
        }

        return as.length - bs.length
      })

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
      this.modal = {
        dn: subdomain,
        step: 1,
        svcData: {},
        svcSelected: null
      }
      this.$bvModal.show('modal-addSvc')
    },

    addNewAlias (subdomain) {
      this.modal = {
        dn: subdomain,
        alias: ''
      }
      this.$bvModal.show('modal-addAlias')
    },

    addSubdomain () {
      this.modal = {
        dn: '',
        step: 0,
        svcData: {},
        svcSelected: null
      }
      this.$bvModal.show('modal-addSvc')
    },

    showServiceWindow (service) {
      this.modal = {
        dn: service._domain,
        svcData: service
      }
      this.$bvModal.show('modal-updSvc')
    },

    goToAnchor () {
      var hash = this.$route.hash.substr(1)
      if (!this.isLoading && hash.length > 0) {
        setTimeout(function () {
          window.scrollTo(0, document.getElementById(hash).offsetTop)
        }, 500)
      }
    },

    handleModalSvcOk (bvModalEvt) {
      bvModalEvt.preventDefault()

      if (this.modal.step === 0 && this.modal.dn !== '') {
        if (this.validateNewSubdomain()) {
          this.modal.step = 1
        }
      } else if (this.modal.step === 1 && this.modal.svcSelected !== null) {
        this.modal.step = 2
      } else if (this.modal.step === 2 && this.modal.svcSelected !== null) {
        ZoneApi
          .addZoneService(this.domain.domain, this.zoneId, this.modal.dn, { Service: this.modal.svcData, _svctype: this.modal.svcSelected })
          .then(
            (response) => {
              this.myServices = response.data
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

    handleUpdateSvc (bvModalEvt) {
      bvModalEvt.preventDefault()

      ZoneApi.updateZoneService(this.domain.domain, this.zoneId, this.modal.svcData)
        .then(
          (response) => {
            this.updateMyServices(response.data)
            this.$nextTick(() => {
              this.$bvModal.hide('modal-updSvc')
            })
          },
          (error) => {
            this.$nextTick(() => {
              this.$bvModal.hide('modal-updSvc')
              this.$bvToast.toast(
                error.response.data.errmsg, {
                  title: 'An error occurs when updating the service!',
                  autoHideDelay: 5000,
                  variant: 'danger',
                  toaster: 'b-toaster-content-right'
                }
              )
            })
          }
        )
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

    validateDomain (dn) {
      var ret = null
      if (dn.length !== 0) {
        ret = dn.length >= 1 && dn.length <= 254

        if (ret) {
          var domains = dn.split('.')

          var newDomainState = ret
          domains.forEach(function (domain) {
            newDomainState &= domain.length >= 1 && domain.length <= 63
            newDomainState &= domain[0] !== '-' && domain[domain.length - 1] !== '-'
            newDomainState &= /^[a-zA-Z0-9]([a-zA-Z0-9-]?[a-zA-Z0-9])*$/.test(domain)
          })
          ret = newDomainState > 0
        }
      }

      return ret
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

      this.modal.newDomainState = this.validateDomain(this.modal.alias)

      return this.modal.newDomainState
    },

    validateNewSubdomain () {
      this.modal.newDomainState = this.validateDomain(this.modal.dn)
      return this.modal.newDomainState
    }
  }
}
</script>
