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
  <b-list-group>
    <b-list-group-item v-if="isLoading" class="d-flex justify-content-center align-items-center">
      <b-spinner variant="primary" label="Spinning" class="mr-3" /> Retrieving your providers...
    </b-list-group-item>
    <b-list-group-item v-if="!isLoading && providers_getAll.length == 0" class="text-center">
      You have no provider defined currently. Try <a href="#" @click.prevent="$emit('new-provider')">adding one</a>!
    </b-list-group-item>
    <b-list-group-item v-for="(provider, index) in sortedProviders" :key="index" :active="mySelectedProvider && mySelectedProvider._id === provider._id" button class="d-flex justify-content-between align-items-center" @click="selectProvider(provider)">
      <div class="d-flex">
        <div class="text-center" style="width: 50px;">
          <img v-if="providerSpecs_getAll && providerSpecs_getAll[provider._srctype]" :src="'/api/providers/_specs/' + provider._srctype + '/icon.png'" :alt="providerSpecs_getAll[provider._srctype].name" :title="providerSpecs_getAll[provider._srctype].name" style="max-width: 100%; max-height: 2.5em; margin: -.6em .4em -.6em -.6em">
        </div>
        <div v-if="provider._comment" style="overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">
          {{ provider._comment }}
        </div>
        <em v-else>No name</em>
      </div>
      <div v-if="!(noLabel && noDropdown)" class="d-flex">
        <div v-if="!noLabel">
          <b-badge class="ml-1" :variant="domain_in_providers[provider._id] > 0 ? 'success' : 'danger'">
            {{ domain_in_providers[provider._id] }} domain(s) associated
          </b-badge>
          <b-badge v-if="providerSpecs_getAll && providerSpecs_getAll[provider._srctype]" class="ml-1" variant="secondary" :title="provider._srctype">
            {{ providerSpecs_getAll[provider._srctype].name }}
          </b-badge>
        </div>
        <b-dropdown v-if="!noDropdown" no-caret size="sm" style="margin-right: -10px" variant="link">
          <template #button-content>
            <b-icon icon="three-dots" />
          </template>
          <b-dropdown-item @click="updateProvider($event, provider)">
            {{ $t('provider.update') }}
          </b-dropdown-item>
          <b-dropdown-item @click="deleteProvider($event, provider)">
            {{ $t('provider.delete') }}
          </b-dropdown-item>
        </b-dropdown>
      </div>
    </b-list-group-item>
  </b-list-group>
</template>

<script>
import { mapGetters } from 'vuex'

import ProviderApi from '@/api/providers'

export default {
  name: 'ProviderList',

  props: {
    emitNewIfEmpty: {
      type: Boolean,
      default: false
    },
    noDropdown: {
      type: Boolean,
      default: false
    },
    noLabel: {
      type: Boolean,
      default: false
    },
    selectedProvider: {
      type: Object,
      default: null
    }
  },

  data: function () {
    return {
      mySelectedProvider: null
    }
  },

  computed: {
    domain_in_providers () {
      const ret = {}

      if (this.providers_getAll != null) {
        for (const i in this.providers_getAll) {
          ret[i] = 0
        }
      }

      if (this.domains_getAll != null) {
        this.domains_getAll.forEach(function (domain) {
          if (!ret[domain.id_provider]) {
            ret[domain.id_provider] = 0
          }
          ret[domain.id_provider]++
        })
      }

      return ret
    },

    isLoading () {
      return (!this.noLabel && this.domains_getAll == null) || this.providers_getAll == null || this.providerSpecs_getAll == null
    },

    ...mapGetters('domains', ['domains_getAll']),
    ...mapGetters('providers', ['sortedProviders', 'providers_getAll']),
    ...mapGetters('providerSpecs', ['providerSpecs_getAll'])
  },

  watch: {
    selectedProvider: function (provider) {
      if (provider !== this.mySelectedProvider && this.providers_getAll[provider._id]) {
        this.selectProvider(provider ? this.providers_getAll[provider._id] : null)
      }
    },

    providers_getAll: function (providers) {
      // handle emitNewIfEmpty
      if (Object.keys(providers).length === 0 && this.emitNewIfEmpty) {
        this.$emit('new-provider')
      }

      // handle case when waiting for providers to select one
      if (this.selectedProvider && this.selectedProvider !== this.mySelectedProvider) {
        this.selectProvider(providers[this.selectedProvider._id])
      }

      // handle deletion of selected provider
      if (this.mySelectedProvider && !providers[this.mySelectedProvider._id]) {
        this.selectProvider(null)
      }
    }
  },

  mounted () {
    if (this.selectedProvider && this.providers_getAll) {
      this.selectProvider(this.providers_getAll[this.selectedProvider._id])
    }
  },

  methods: {
    deleteProvider (event, provider) {
      event.stopPropagation()
      ProviderApi.deleteProvider(provider)
        .then(
          response => {
            this.$store.dispatch('providers/getAllMyProviders')
            this.$bvToast.toast(
              'The provider has been deleted with success.', {
                title: 'Provider deleted with success',
                autoHideDelay: 5000,
                variant: 'success',
                toaster: 'b-toaster-content-right'
              })
          },
          error => {
            this.$bvToast.toast(
              error.response.data.errmsg, {
                title: 'An error occurs when trying to delete the provider:',
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              })
          })
    },

    selectProvider (provider) {
      if (this.mySelectedProvider != null && provider && this.mySelectedProvider._id === provider._id) {
        this.mySelectedProvider = null
      } else {
        this.mySelectedProvider = provider
      }
      this.$emit('provider-selected', this.mySelectedProvider)
    },

    updateProvider (event, provider) {
      event.stopPropagation()
      this.$router.push('/providers/' + encodeURIComponent(provider._id))
    }
  }
}
</script>
