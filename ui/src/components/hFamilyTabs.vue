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
  <b-tabs :content-class="contentClass">
    <b-tab title="All" active>
      <b-list-group>
        <b-list-group-item v-for="(svc, idx) in availableNewServices()" :key="idx" :active="value === idx" button class="d-flex" @click="$emit('input', idx)">
          <div v-if="svc._svcicon" class="d-inline-block align-self-center text-center" style="width: 75px;">
            <img :src="svc._svcicon" :alt="svc.name" style="max-width: 100%; max-height: 2.5em; margin: -.6em .4em -.6em -.6em">
          </div>
          <div>
            {{ svc.name }}
            <small class="text-muted">{{ svc.description }}</small>
            <b-badge v-for="(categorie, idcat) in svc.categories" :key="idcat" variant="gray" class="float-right ml-1">
              {{ categorie }}
            </b-badge>
          </div>
        </b-list-group-item>
        <b-list-group-item v-for="(svc, idx) in disabledNewServices()" :key="idx" :active="value === idx" class="d-flex" disabled @click="$emit('input', idx)">
          <div v-if="svc._svcicon" class="d-inline-block align-self-center text-center" style="width: 75px;">
            <img :src="svc._svcicon" :alt="svc.name" style="max-width: 100%; max-height: 2.5em; margin: -.6em .4em -.6em -.6em">
          </div>
          <div>
            <span :title="svc.description">{{ svc.name }}</span> <small class="font-italic text-danger">{{ filteredNewServices[idx] }}</small>
            <b-badge v-for="(categorie, idcat) in svc.categories" :key="idcat" variant="gray" class="float-right ml-1">
              {{ categorie }}
            </b-badge>
          </div>
        </b-list-group-item>
      </b-list-group>
    </b-tab>
    <b-tab v-for="(family, idxf) in families" :key="idxf" :title="family.label">
      <b-list-group>
        <b-list-group-item v-for="(svc, idx) in availableNewServices(family.family)" :key="idx" :active="value === idx" button class="d-flex" @click="$emit('input', idx)">
          <div v-if="svc._svcicon" class="d-inline-block align-self-center text-center" style="width: 75px;">
            <img :src="svc._svcicon" :alt="svc.name" style="max-width: 100%; max-height: 2.5em; margin: -.6em .4em -.6em -.6em">
          </div>
          <div>
            {{ svc.name }}
            <small class="text-muted">{{ svc.description }}</small>
            <b-badge v-for="(categorie, idcat) in svc.categories" :key="idcat" variant="gray" class="float-right ml-1">
              {{ categorie }}
            </b-badge>
          </div>
        </b-list-group-item>
        <b-list-group-item v-for="(svc, idx) in disabledNewServices(family.family)" :key="idx" :active="value === idx" class="d-flex" disabled @click="$emit('input', idx)">
          <div v-if="svc._svcicon" class="d-inline-block align-self-center text-center" style="width: 75px;">
            <img :src="svc._svcicon" :alt="svc.name" style="max-width: 100%; max-height: 2.5em; margin: -.6em .4em -.6em -.6em">
          </div>
          <div>
            <span :title="svc.description">{{ svc.name }}</span> <small class="font-italic text-danger">{{ filteredNewServices[idx] }}</small>
            <b-badge v-for="(categorie, idcat) in svc.categories" :key="idcat" variant="gray" class="float-right ml-1">
              {{ categorie }}
            </b-badge>
          </div>
        </b-list-group-item>
      </b-list-group>
    </b-tab>
  </b-tabs>
</template>

<script>
import { mapGetters } from 'vuex'

export default {
  name: 'HFamilyTabs',

  props: {
    contentClass: {
      type: String,
      default: ''
    },
    domain: {
      type: Object,
      required: true
    },
    dn: {
      type: String,
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
    value: {
      type: String,
      default: ''
    }
  },

  data: function () {
    return {
      availableResourceTypes: [],
      families: [
        {
          label: 'Services',
          family: 'abstract'
        },
        {
          label: 'Providers',
          family: 'provider'
        },
        {
          label: 'Raw DNS resources',
          family: ''
        }
      ]
    }
  },

  computed: {
    isLoading () {
      return this.availableResourceTypes.length === 0
    },

    filteredNewServices () {
      return this.analyzeRestrictions(this.services, this.availableResourceTypes)
    },

    ...mapGetters('providers', ['providers_getAll']),
    ...mapGetters('providerSpecs', ['providerSpecs_getAll'])
  },

  watch: {
    providers_getAll: function (t) {
      this.loadProviderSpecs(t)
    },
    providerSpecs_getAll: function (t) {
      this.loadProviderSpecs(t)
    }
  },

  created () {
    if (this.providerSpecs_getAll !== null && this.providers_getAll !== null) {
      this.loadProviderSpecs(this.providerSpecs_getAll)
    }
  },

  methods: {
    loadProviderSpecs (specs) {
      if (this.providerSpecs_getAll === null || this.providers_getAll === null) {
        return
      }

      const availableResourceTypes = []

      specs[this.providers_getAll[this.domain.id_provider]._srctype].capabilities.forEach(function (i) {
        if (i.startsWith('rr-')) {
          availableResourceTypes.push(parseInt(i.substr(3, i.indexOf('-', 4) - 3)))
        }
      })

      this.availableResourceTypes = availableResourceTypes
    },

    analyzeRestrictions (allServices, availableResourceTypes) {
      const ret = {}

      for (const type in allServices) {
        const svc = allServices[type]

        if (svc.restrictions) {
          // Handle NeedTypes restriction: hosting provider need to support given types.
          if (svc.restrictions.needTypes) {
            for (const k in svc.restrictions.needTypes) {
              if (availableResourceTypes.indexOf(svc.restrictions.needTypes[k]) < 0) {
                ret[type] = 'is not available on this domain name hosting provider.'
                continue
              }
            }
          }

          let found = false

          // Handle Alone restriction: only nearAlone are allowed.
          if (svc.restrictions.alone && this.myServices.services[this.dn]) {
            found = false
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
              if (found != null) {
                break
              }
            }
            if (found) {
              ret[type] = 'cannot be present along with ' + allServices[found].name + '.'
              continue
            }
          }

          // Check reverse Exclusivity
          found = null
          for (const k in this.myServices.services[this.dn]) {
            const s = this.services[this.myServices.services[this.dn][k]._svctype]
            if (!s.restrictions || !s.restrictions.exclusive) {
              continue
            }
            for (const i in s.restrictions.exclusive) {
              if (svc._svctype === s.restrictions.exclusive[i]) {
                found = s._svctype
                break
              }
            }
            if (found != null) {
              break
            }
          }
          if (found) {
            ret[type] = 'cannot be present along with ' + allServices[found].name + '.'
            continue
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
          let oneAlone = null
          let oneLeaf = null
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

    availableNewServices (family) {
      const ret = {}

      for (const type in this.services) {
        if (this.filteredNewServices[type] == null && (family === undefined || family === this.services[type].family)) {
          ret[type] = this.services[type]
        }
      }

      return ret
    },

    disabledNewServices (family) {
      const ret = {}

      for (const type in this.services) {
        if (this.filteredNewServices[type] != null && (family === undefined || family === this.services[type].family)) {
          ret[type] = this.services[type]
        }
      }

      return ret
    }
  }
}
</script>
