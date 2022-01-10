<!--
    Copyright or © or Copr. happyDNS (2020)

    contact@happydomain.org

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
  <b-container class="d-flex flex-column mt-4" fluid>
    <h1 class="text-center mb-3">
      <button type="button" class="btn font-weight-bolder" @click="$router.go(-1)">
        <b-icon icon="chevron-left" />
      </button>
      {{ $t('wait.updating') }} <em v-if="myProvider">{{ myProvider._comment }}</em>
    </h1>
    <hr class="mt-0 mb-0">

    <b-row class="flex-grow-1">
      <b-col v-if="providerSpecsSelected && providerSpecs_getAll" lg="4" md="5" class="bg-light">
        <div class="text-center mb-3">
          <img :src="'/api/providers/_specs/' + providerSpecsSelected + '/icon.png'" :alt="providerSpecs_getAll[providerSpecsSelected].name" style="max-width: 100%; max-height: 10em">
        </div>
        <h3>
          {{ providerSpecs_getAll[providerSpecsSelected].name }}
        </h3>

        <p class="text-muted text-justify">
          {{ providerSpecs_getAll[providerSpecsSelected].description }}
        </p>

        <div class="text-center mb-2">
          <router-link v-if="providerSpecs_getAll[providerSpecsSelected] && providerSpecs_getAll[providerSpecsSelected].capabilities && providerSpecs_getAll[providerSpecsSelected].capabilities.indexOf('ListDomains') > -1" type="button" variant="secondary" class="btn btn-secondary mb-1" :to="'/providers/' + myProvider._id + '/domains'">
            <b-icon icon="list-task" />
            {{ $t('domains.list') }}
          </router-link>
          <b-button type="button" variant="danger" class="mb-1" @click="deleteProvider()">
            <b-icon icon="trash-fill" />
            {{ $t('provider.delete') }}
          </b-button>
        </div>
      </b-col>

      <b-col lg="8" md="7">
        <b-form v-if="!isLoading" class="mt-2 mb-5" @submit.stop.prevent="nextState">
          <h-provider-state
            v-if="form"
            v-model="settings"
            class="mt-2 mb-2"
            :form="form"
            :provider-name="providerSpecs_getAll[providerSpecsSelected].name"
            :state="state"
          />

          <hr>

          <h-provider-state-buttons v-if="form" class="d-flex justify-content-end" edit :form="form" :next-is-working="nextIsWorking" :previous-is-working="previousIsWorking" @previous-state="previousState" />
        </b-form>
      </b-col>
    </b-row>
  </b-container>
</template>

<script>
import ProvidersApi from '@/api/providers'
import ProviderState from '@/mixins/providerState'
import { mapGetters } from 'vuex'

export default {

  components: {
    hProviderState: () => import('@/components/hProviderState'),
    hProviderStateButtons: () => import('@/components/hProviderStateButtons')
  },

  mixins: [ProviderState],

  data: function () {
    return {
      myProvider: null,
      providerSpecsSelected: null
    }
  },

  computed: {
    isLoading () {
      return this.myProvider == null || this.providerSpecs_getAll == null || this.providerSpecsSelected == null || this.settings == null || this.providerState_isLoading
    },

    ...mapGetters('providerSpecs', ['providerSpecs_getAll'])
  },

  mounted () {
    this.resetSettings()
    ProvidersApi.getProvider(this.$route.params.provider)
      .then(response => {
        this.myProvider = response.data
        this.settings = this.myProvider
        this.providerSpecsSelected = this.myProvider._srctype
        this.loadState(0)
        return true
      })
  },

  methods: {
    deleteProvider () {
      ProvidersApi.deleteProvider(this.$route.params.provider)
        .then(
          response => {
            this.$router.push('/providers/')
          },
          error => {
            this.$root.$bvToast.toast(
              error.response.data.errmsg, {
                title: this.$t('errors.provider-delete'),
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
          })
    },
    reactOnSuccess (toState, newProvider) {
      if (newProvider) {
        this.myProvider = newProvider
      }
    }
  }
}
</script>
