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
  <b-container fluid class="mt-4">
    <h1 class="text-center mb-4">
      <button type="button" class="btn font-weight-bolder" @click="$router.go(-1)">
        <b-icon icon="chevron-left" />
      </button>
      <i18n path="provider.select-provider" tag="span">
        <span class="text-monospace">{{ $route.params.domain }}</span>
      </i18n>
    </h1>

    <div v-if="validatingNewDomain" class="d-flex justify-content-center align-items-center">
      <b-spinner variant="primary" label="Spinning" class="mr-3" /> {{ $t('wait.validating') }};
    </div>

    <b-row v-else>
      <b-col offset-md="2" md="8">
        <provider-list ref="providerList" emit-new-if-empty @new-provider="newProvider" @provider-selected="addDomainToProvider($event, $route.params.domain, true)" />

        <p class="text-center mt-3">
          {{ $t('provider.find') }} <a href="#" @click.prevent="newProvider">{{ $t('domains.add-now') }}</a>
        </p>
      </b-col>
    </b-row>

    <h-modal-add-provider ref="addSrcModal" @update-my-providers="doneAdd" />
  </b-container>
</template>

<script>
import AddDomainToProvider from '@/mixins/addDomainToProvider'

export default {

  components: {
    hModalAddProvider: () => import('@/components/hModalAddProvider'),
    providerList: () => import('@/components/providerList')
  },

  mixins: [AddDomainToProvider],

  methods: {
    doneAdd () {
      this.$refs.providerList.updateProviders()
    },

    newProvider () {
      this.$refs.addSrcModal.show()
    }
  }
}
</script>
