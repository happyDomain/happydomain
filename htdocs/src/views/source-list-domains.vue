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
  <b-container class="mt-4 mb-5">
    <i18n path="source.source" tag="h3" class="text-center mb-4">
      <em v-if="mySource">{{ mySource._comment }}</em>
    </i18n>
    <h-source-list-domains ref="dlist" show-already-imported :source="mySource" />
    <h-list-group-input-new-domain v-if="$refs.dlist && $refs.dlist.noDomainsList" autofocus class="mt-2" />
  </b-container>
</template>

<script>
export default {

  components: {
    hListGroupInputNewDomain: () => import('@/components/hListGroupInputNewDomain'),
    hSourceListDomains: () => import('@/components/hSourceListDomains')
  },

  props: {
    mySource: {
      type: Object,
      required: true
    }
  },

  created () {
    this.$store.dispatch('domains/getAllMyDomains')
    this.$store.dispatch('sources/getAllMySources')
  }
}
</script>
