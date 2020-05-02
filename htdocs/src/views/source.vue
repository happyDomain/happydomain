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

      <div class="text-center">
        <b-button type="button" variant="secondary" @click="showListImportableDomain()" v-if="source_specs && source_specs.capabilities && source_specs.capabilities.indexOf('ListDomains') > -1">
          <b-icon icon="list-task" />
          List importable domains
        </b-button>
      </div>
    </b-col>

    <b-col lg="8" md="7">
      <router-view :parentLoading="isLoading" :mySource="mySource" :sources="sources" :source_specs="source_specs" :source_specs_selected="source_specs_selected"></router-view>
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
      sources: null,
      source_specs: null,
      source_specs_selected: null
    }
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
    showListImportableDomain () {
      this.$router.push('/sources/' + this.$route.params.source + '/domains')
    }
  }
}
</script>
