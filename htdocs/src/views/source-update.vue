<template>
<b-container fluid class="mt-4">

  <h1 class="text-center mb-4">
    <button type="button" @click="$router.go(-1)" class="btn font-weight-bolder"><b-icon icon="chevron-left"></b-icon></button>
    Updating your domain name source <em v-if="mySource">{{ mySource._comment }}</em>
  </h1>
  <hr style="margin-bottom:0">

  <b-row>
    <b-col lg="4" md="5" style="background-color: #EAFFEC" v-if="source_specs_selected && sources">
      <div class="text-center mb-3">
        <img :src="'/api/source_specs/' + source_specs_selected + '.png'" :alt="sources[source_specs_selected].name" style="max-width: 100%; max-height: 10em">
      </div>
      <h3>
        {{ sources[source_specs_selected].name }}
      </h3>

      <p class="text-muted text-justify">{{ sources[source_specs_selected].description }}</p>
    </b-col>

    <b-col lg="8" md="7">
      <form @submit.stop.prevent="submitSource" v-if="!isLoading" class="mt-2 mb-5">

        <b-form-group
          id="input-spec-name"
          label="Name your source:"
          label-for="source-name"
          description="Give an explicit name in order to easily find this service."
          >
          <b-form-input
            id="source-name"
            v-model="mySource._comment"
            required
            :placeholder="sources[source_specs_selected].name + ' 1'"
            ></b-form-input>
        </b-form-group>

        <b-form-group
          v-for="(spec, index) in source_specs"
          v-bind:key="index"
          :id="'input-spec-' + index"
          :label="spec.label"
          :label-for="'spec-' + index"
          :description="spec.description"
          >
          <b-form-input
            :id="'spec-' + index"
            v-model="mySource.Source[spec.id]"
            :required="spec.required !== undefined && spec.required"
            :placeholder="spec.placeholder"
            v-if="spec.choices === undefined"
            ></b-form-input>
          <b-form-select
            :id="'spec-' + index"
            v-model="mySource.Source[spec.id]"
            :required="spec.required !== undefined && spec.required"
            :options="spec.choices"
            v-if="spec.choices !== undefined"
            ></b-form-select>
        </b-form-group>

        <div class="ml-3 mr-3">
          <b-button type="button" variant="secondary" @click="$router.go(-1)">&lt; Cancel</b-button>
          <b-button class="float-right" type="submit" variant="primary">Update this source &gt;</b-button>
        </div>
      </form>
    </b-col>
  </b-row>
</b-container>
</template>

<script>
import axios from 'axios'

export default {

  data: function () {
    return {
      mySource: null,
      prevRoute: null,
      sources: null,
      source_specs: null,
      source_specs_selected: null
    }
  },

  beforeRouteEnter (to, from, next) {
    next(vm => {
      vm.prevRoute = from
    })
  },

  mounted () {
    axios
      .get('/api/sources/' + this.$route.params.source)
      .then(response => {
        this.mySource = response.data
        this.source_specs_selected = this.mySource._srctype

        axios
          .get('/api/source_specs/' + this.mySource._srctype)
          .then(response => {
            this.source_specs = response.data
            return true
          })

        return true
      })
    axios
      .get('/api/source_specs')
      .then(response => {
        this.sources = response.data
        return true
      })
  },

  computed: {
    isLoading () {
      return this.mySource == null || this.sources == null || this.source_specs == null || this.source_specs_selected == null
    }
  },

  methods: {

    submitSource () {
      axios
        .put('/api/sources/' + this.mySource._id, this.mySource)
        .then(
          (response) => {
            this.$bvToast.toast(
              'Great! ' + response.data.domain + ' has been added. You can manage it right now.', {
                title: 'New domain attached to happyDNS!',
                autoHideDelay: 5000,
                variant: 'success',
                href: 'domains/' + response.data.domain,
                toaster: 'b-toaster-content-right'
              }
            )
            this.$router.push(this.prevRoute)
          },
          (error) => {
            console.log(error.data)
            this.$bvToast.toast(
              error.data.errmsg, {
                title: 'An error occurs when creating the source!',
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

<style>
  .form-group label {
    font-weight: bold;
  }
</style>
