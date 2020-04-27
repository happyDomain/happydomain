<template>
<b-container class="mt-4">

  <h1 class="text-center mb-4">Select the source where lives your domain <code>{{ $route.params.domain }}</code></h1>

  <div v-if="step === 0">
    <h3 v-if="mySources.length > 0">Existing sources</h3>

    <div class="d-flex flex-row justify-content-around flex-wrap align-self-center" v-if="!loading && mySources.length > 0">
      <div type="button" @click="selectExistingSource(src)" class="p-3 source" v-for="(src, index) in mySources" v-bind:key="index">
        <img :src="sources[src['_srctype']].icon" :alt="sources[src['_srctype']].name">
        Utiliser {{ src.comment }}
      </div>
    </div>

    <h3 v-if="!loading">New source</h3>

    <div class="d-flex flex-row justify-content-around flex-wrap align-self-center" v-if="!loading">
      <div type="button" @click="selectNewSource(index)" class="p-3 source" v-for="(src, index) in sources" v-bind:key="index">
        <img :src="src.icon" :alt="src.icon">
        Utiliser {{ src.name }}<br>
        <p class="text-muted">
          {{ src.description }}
        </p>
      </div>
    </div>
  </div>

  <div v-if="step === 1">
    <b-row>
      <b-col lg="4" md="5">
        <h3>
          {{ sources[source_specs_selected].name }}
        </h3>

        <p class="text-muted text-justify">{{ sources[source_specs_selected].description }}</p>
      </b-col>

      <b-col lg="8" md="7">
        <form @submit.stop.prevent="submitNewSource" v-if="!loading">

          <b-form-group
            id="input-spec-name"
            label="Name your source:"
            label-for="source-name"
            description="Give an explicit name in order to easily find this service."
            >
            <b-form-input
              id="source-name"
              v-model="new_source_name"
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
              v-model="spec.value"
              :required="spec.required !== undefined && spec.required"
              :placeholder="spec.placeholder"
              v-if="spec.choices === undefined"
              ></b-form-input>
            <b-form-select
              :id="'spec-' + index"
              v-model="spec.value"
              :required="spec.required !== undefined && spec.required"
              :options="spec.choices"
              v-if="spec.choices !== undefined"
              ></b-form-select>
          </b-form-group>

          <div class="ml-3 mr-3">
            <b-button type="button" variant="secondary" @click="step--">&lt; Use another source</b-button>
            <b-button class="float-right" type="submit" variant="primary">Add this source &gt;</b-button>
          </div>
        </form>
      </b-col>
    </b-row>
  </div>

</b-container>
</template>

<script>
import axios from 'axios'

export default {

  data: function () {
    return {
      loading: true,
      mySources: [],
      new_source_name: '',
      sources: {},
      source_specs: {},
      source_specs_selected: null,
      step: 0
    }
  },

  mounted () {
    axios
      .get('/api/sources')
      .then(response => { this.mySources = response.data; this.loading = Object.keys(this.sources).length === 0; return true })
    axios
      .get('/api/source_specs')
      .then(response => { this.sources = response.data; this.loading = false; return true })
  },

  methods: {

    selectNewSource (sourceSpec) {
      this.step = 1
      this.loading = true
      this.source_specs_selected = sourceSpec
      axios
        .get('/api/source_specs/' + sourceSpec)
        .then(response => { this.source_specs = response.data; this.loading = false; return true })
    },

    selectExistingSource (source) {
      console.log(source)
      this.step = 2
      this.loading = true

      axios
        .post('/api/domains', {
          id_source: source._id,
          domain: this.$route.params.domain
        })
        .then(
          (response) => {
            console.log(response.data)
            this.$bvToast.toast(
              'Great! ' + response.data.domain + ' has been added. You can manage it right now.', {
                title: 'New domain attached to happyDNS!',
                autoHideDelay: 5000,
                variant: 'success',
                href: 'domains/' + response.data.domain,
                toaster: 'b-toaster-content-right'
              }
            )
            this.$router.push('/')
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
    },

    submitNewSource () {
      var mySource = {
        _srctype: this.source_specs_selected,
        _comment: this.new_source_name,
        Source: {}
      }

      this.source_specs.forEach(function (spec) {
        if (spec.value) {
          mySource.Source[spec.id] = spec.value
        } else if (spec.default) {
          mySource.Source[spec.id] = spec.default
        }
      })

      console.log(mySource)

      axios
        .post('/api/sources', mySource)
        .then(
          (response) => {
            this.selectExistingSource(response.data)
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
.source {
    box-shadow: 2px 2px black;
    border: 1px solid black;
    display: flex;
    flex-direction: column;
    align-items: center;
    margin: 2.5%;
    width: 20%;
    height: 150px;
    text-align: center;
    vertical-align: middle;
}
.source img {
    max-width: 100%;
    max-height: 90%;
    padding: 2%;
}
</style>
