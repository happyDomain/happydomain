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
  <b-container fluid class="mt-4">
    <h1 class="text-center mb-4">
      <button type="button" class="btn font-weight-bolder" @click="$router.go(-1)">
        <b-icon icon="chevron-left" />
      </button>
      Select the source where lives your domain <span class="text-monospace">{{ $route.params.domain }}</span>
    </h1>

    <div v-if="validating" class="d-flex justify-content-center align-items-center">
      <b-spinner variant="primary" label="Spinning" class="mr-3" /> Validating domain &hellip;
    </div>

    <b-row v-else>
      <b-col offset-md="2" md="8">
        <source-list ref="sourceList" emit-new-if-empty @newSource="newSource" @sourceSelected="selectExistingSource" />

        <p class="text-center mt-3">
          Can't find the source here? <a href="#" @click.prevent="newSource">Add it now!</a>
        </p>
      </b-col>
    </b-row>

    <h-modal-add-source ref="addSrcModal" @done="doneAdd" />
  </b-container>
</template>

<script>
import axios from 'axios'

export default {

  components: {
    hModalAddSource: () => import('@/components/hModalAddSource'),
    sourceList: () => import('@/components/sourceList')
  },

  data: function () {
    return {
      validating: false
    }
  },

  methods: {
    doneAdd () {
      this.$refs.sourceList.updateSources()
    },

    newSource () {
      this.$refs.addSrcModal.show()
    },

    selectExistingSource (source) {
      this.validating = true

      axios
        .post('/api/domains', {
          id_source: source._id,
          domain: this.$route.params.domain
        })
        .then(
          (response) => {
            this.$root.$bvToast.toast(
              'Great! ' + response.data.domain + ' has been added. You can manage it right now.', {
                title: 'New domain attached to happyDNS!',
                autoHideDelay: 5000,
                variant: 'success',
                toaster: 'b-toaster-content-right'
              }
            )
            this.$router.push('/domains/' + encodeURIComponent(response.data.domain))
          },
          (error) => {
            this.validating = false
            this.$bvToast.toast(
              error.response.data.errmsg, {
                title: 'An error occurs when adding the domain!',
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
          }
        )
    }
  }
}
</script>
