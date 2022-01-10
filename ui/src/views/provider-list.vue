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
  <b-container class="pt-4 pb-5">
    <b-button type="button" variant="primary" class="float-right" @click="newProvider">
      <b-icon icon="plus" />
      {{ $t('common.add-new-thing', { thing: $t('provider.kind') }) }}
    </b-button>
    <h1 class="text-center mb-4">
      {{ $t('provider.title') }}
    </h1>
    <b-row>
      <b-col offset-md="2" md="8">
        <provider-list @new-provider="newProvider" @provider-selected="click" />
      </b-col>
    </b-row>
  </b-container>
</template>

<script>
export default {

  components: {
    providerList: () => import('@/components/providerList')
  },

  created () {
    this.$store.dispatch('domains/getAllMyDomains')
    this.$store.dispatch('providers/getAllMyProviders')
  },

  methods: {
    click (provider) {
      this.$router.push('/providers/' + encodeURIComponent(provider._id))
    },

    newProvider () {
      this.$router.push('/providers/new')
    }
  }
}
</script>
