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
