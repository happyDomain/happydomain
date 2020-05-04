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
<form @submit.stop.prevent="submitSource" v-if="!isLoading" class="mt-2 mb-5">
  <div class="float-right">
    <b-button type="button" variant="outline-primary" @click="edit=true" v-if="!edit">
      <b-icon icon="pencil" />
      Edit
    </b-button>
    <b-button type="button" variant="primary" @click="submitSource()" v-else>
      <b-icon icon="check" />
      Update this source
    </b-button>
  </div>

  <b-form-group
    id="input-spec-name"
    label="Source's name"
    label-for="source-name"
    :description="edit?'Give an explicit name in order to easily find this service.':''"
    >
    <b-form-input
      id="source-name"
      v-model="mySource._comment"
      required
      :placeholder="sources[source_specs_selected].name + ' 1'"
      :plaintext="!edit"
      ></b-form-input>
  </b-form-group>

  <hr>

  <b-form-group
    v-for="(spec, index) in source_specs.fields"
    v-bind:key="index"
    :id="'input-spec-' + index"
    :label="spec.label"
    :label-for="'spec-' + index"
    :description="edit?spec.description:''"
    v-show="edit || !spec.secret"
    >
    <b-form-input
      :id="'spec-' + index"
      v-model="mySource.Source[spec.id]"
      :required="spec.required !== undefined && spec.required"
      :placeholder="spec.placeholder"
      :plaintext="!edit"
      v-if="!edit || spec.choices === undefined"
      ></b-form-input>
    <b-form-select
      :id="'spec-' + index"
      v-model="mySource.Source[spec.id]"
      :required="spec.required !== undefined && spec.required"
      :options="spec.choices"
      v-if="edit && spec.choices !== undefined"
      ></b-form-select>
  </b-form-group>
</form>
</template>

<script>
import axios from 'axios'

export default {

  data: function () {
    return {
      edit: false,
      prevRoute: null
    }
  },

  beforeRouteEnter (to, from, next) {
    next(vm => {
      vm.prevRoute = from
    })
  },

  computed: {
    isLoading () {
      return this.parentLoading
    }
  },

  methods: {

    submitSource () {
      axios
        .put('/api/sources/' + this.mySource._id, this.mySource)
        .then(
          (response) => {
            this.$root.$bvToast.toast(
              'Great! ' + response.data.domain + ' has been updated!', {
                title: 'Source updated successfully',
                autoHideDelay: 5000,
                variant: 'success',
                toaster: 'b-toaster-content-right'
              }
            )
            this.$router.push(this.prevRoute)
          },
          (error) => {
            this.$bvToast.toast(
              error.response.data.errmsg, {
                title: 'An error occurs when creating the source!',
                autoHideDelay: 5000,
                variant: 'danger',
                toaster: 'b-toaster-content-right'
              }
            )
          }
        )
    }

  },

  props: ['parentLoading', 'mySource', 'sources', 'source_specs', 'source_specs_selected']
}
</script>

<style>
  .form-group label {
    font-weight: bold;
  }
</style>
