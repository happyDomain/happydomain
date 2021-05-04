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
  <div ng-if="!isLoading()">
    <div v-for="(v,key) in value" :key="key">
      <b-row>
        <b-col>
          <h3>
            <span v-if="!editKeys[key]">{{ key }}</span>
            <b-input-group v-else>
              <b-form-input
                v-model="newKeys[key]"
                :placeholder="specs.id + ' key name'"
              />
              <template #append>
                <b-button v-if="editKeys[key]" type="button" size="sm" variant="primary" @click="rename(key)">
                  <b-icon icon="check" />
                  <span v-if="key">{{ $t('common.rename') }}</span>
                  <span v-else>{{ $t('domains.create-new-key', { id: specs.id }) }}</span>
                </b-button>
              </template>
            </b-input-group>
            <b-button v-if="!editKeys[key]" type="button" size="sm" variant="link" @click="toogleKeyEdit(key)">
              <b-icon icon="pencil" />
            </b-button>
          </h3>
        </b-col>
        <b-col v-if="!key || !editKeys[key]" sm="auto">
          <b-button v-if="!editChildrenKeys[key]" type="button" size="sm" variant="outline-primary" class="mx-1" @click="toogleChildrenEdit(key)">
            <b-icon icon="pencil" />
            {{ $t('common.edit') }}
          </b-button>
          <b-button v-else type="button" size="sm" variant="success" class="mx-1" @click="saveObject(key)">
            <b-icon icon="check" />
            {{ $t('domains.save-modifications') }}
          </b-button>
          <b-button type="button" size="sm" variant="outline-danger" class="mx-1" @click="deleteKey(key)">
            <b-icon icon="trash" />
            {{ $t('common.delete') }}
          </b-button>
        </b-col>
        <b-col v-else sm="auto">
          <b-button type="button" variant="outline-danger" class="mx-1" @click="cancelKeyEdit(key)">
            <b-icon icon="x-circle" />
            {{ $t('common.cancel-edit') }}
          </b-button>
        </b-col>
      </b-row>
      <h-resource-value
        v-if="key"
        v-model="v[key]"
        :edit="editChildrenKeys[key]"
        :services="services"
        :specs="service_specs"
        :type="main_type"
        @save-service="$emit('save-service')"
      />
      <hr>
    </div>
    <b-button @click="createKey()">
      {{ $t('common.add-new-thing', { thing: specs.id }) }}
    </b-button>
  </div>
</template>

<script>
import ServiceSpecsApi from '@/services/ServiceSpecsApi'
import Vue from 'vue'

export default {
  name: 'HResourceValueMap',

  components: {
    hResourceValue: () => import('@/components/hResourceValue')
  },

  props: {
    services: {
      type: Object,
      required: true
    },
    specs: {
      type: Object,
      default: null
    },
    type: {
      type: String,
      required: true
    },
    value: {
      type: Object,
      required: true
    }
  },

  data: function () {
    return {
      editChildrenKeys: {},
      editKeys: {},
      newKeys: {},
      key_type: '',
      main_type: '',
      service_specs: null
    }
  },

  computed: {
    isLoading () {
      return this.service_specs == null
    },

    fieldsNames () {
      const ret = []
      this.service_specs.fields.forEach(function (sspec) {
        ret.push({
          key: sspec.id,
          sortable: true
        })
      })
      return ret
    },

    val: {
      get () {
        return this.value
      },
      set (value) {
        this.$emit('input', value)
      }
    }
  },

  watch: {
    type: function () {
      this.key_type = this.type.substr(this.type.indexOf('[') + 1, this.type.indexOf(']') - this.type.indexOf('[') - 1)
      this.main_type = this.type.substr(this.type.indexOf(']') + 1)
      this.pullServiceSpecs()
    }
  },

  created () {
    if (this.type !== undefined) {
      this.key_type = this.type.substr(this.type.indexOf('[') + 1, this.type.indexOf(']') - this.type.indexOf('[') - 1)
      this.main_type = this.type.substr(this.type.indexOf(']') + 1)
      this.pullServiceSpecs()
    }
  },

  methods: {
    pullServiceSpecs () {
      ServiceSpecsApi.getServiceSpecs(this.main_type)
        .then(
          (response) => {
            this.service_specs = response.data
          }
        )
    },

    cancelKeyEdit (key) {
      Vue.delete(this.editKeys, key)
    },

    createKey () {
      let newObj = {}
      if (this.main_type.substr(0, 2) === '[]') {
        newObj = []
      }
      Vue.set(this.value, '', newObj)
      this.toogleKeyEdit('', true)
    },

    deleteKey (key) {
      Vue.delete(this.value, key)
      if (Object.keys(this.value).length === 0) {
        this.$emit('input', undefined)
        this.$emit('save-service')
      }
    },

    rename (key) {
      if (key !== this.newKeys[key]) {
        Vue.set(this.value, this.newKeys[key], this.value[key])
        Vue.delete(this.value, key)
      }
      Vue.delete(this.editKeys, key)
      this.$emit('save-service')
    },

    saveChildrenValues () {
      for (const key in this.editChildrenKeys) {
        if (this.editChildrenKeys[key]) {
          this.saveObject(key)
        }
      }

      for (const key in this.editKeys) {
        if (this.editKeys[key]) {
          this.rename(key)
        }
      }
    },

    saveObject (key) {
      const vm = this
      this.$emit('save-service', function () {
        Vue.set(vm.editChildrenKeys, key, false)
      })
    },

    toogleChildrenEdit (key) {
      Vue.set(this.editChildrenKeys, key, !this.editChildrenKeys[key])
    },

    toogleKeyEdit (key, forceValue) {
      Vue.set(this.newKeys, key, key)
      if (forceValue == null) {
        forceValue = !this.editKeys[key]
      }
      Vue.set(this.editKeys, key, forceValue)
    }
  }
}
</script>
