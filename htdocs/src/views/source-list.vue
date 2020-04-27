<template>
<b-container class="mt-4">
  <b-button type="button" @click="newSource" variant="primary" class="float-right">
    <b-icon icon="plus"></b-icon>
    Add new source
  </b-button>
  <h1 class="text-center mb-4">
    Your sources
  </h1>
  <b-row>
    <b-col offset-md="2" md="8">
      <div class="text-right">
      </div>
      <b-list-group>
        <b-list-group-item v-if="loading" class="text-center">
          <b-spinner variant="secondary" label="Spinning"></b-spinner> Retrieving your sources...
        </b-list-group-item>
        <b-list-group-item :to="'/sources/' + source._id" v-for="(source, index) in sources" v-bind:key="index" class="d-flex justify-content-between align-items-center">
          <div>
            <img :src="'/api/source_specs/' + source._srctype + '.png'" :alt="sources_specs[source._srctype].name" :title="sources_specs[source._srctype].name" style="max-width: 100%; max-height: 2.5em; margin: -.6em .4em -.6em -.6em">
            <span v-if="source._comment">{{ source._comment }}</span>
            <em v-else>No name</em>
          </div>
          <div>
            <b-badge class="ml-1" :variant="domain_in_sources[index] > 0 ? 'success' : 'danger'">{{ domain_in_sources[index] }} domain(s) associated</b-badge>
            <b-badge class="ml-1" variant="secondary" :title="source._srctype">{{ sources_specs[source._srctype].name }}</b-badge>
          </div>
        </b-list-group-item>
      </b-list-group>
    </b-col>
  </b-row>
</b-container>
</template>

<script>
import axios from 'axios'

export default {

  data: function () {
    return {
      domains: null,
      loading: true,
      sources: null,
      sources_specs: null
    }
  },

  mounted () {
    setTimeout(() => {
      axios
        .get('/api/domains')
        .then(response => { this.domains = response.data; return true })
      axios
        .get('/api/sources')
        .then(response => { this.sources = response.data; return true })
    }, 100)
    axios
      .get('/api/source_specs')
      .then(response => { this.sources_specs = response.data; return true })
  },

  computed: {
    domain_in_sources () {
      var ret = {}

      if (this.domains != null && this.sources != null) {
        this.sources.forEach(function (source, idx) {
          ret[idx] = 0
          this.domains.forEach(function (domain) {
            if (domain.id_source === source._id) {
              ret[idx]++
            }
          })
        }, this)
      }

      return ret
    }
  },

  methods: {
    checkLoading () {
      this.loading = this.domains == null || this.sources == null || this.sources_specs == null
      return this.loading
    },

    show (source) {
      this.$router.push('/sources/' + source.source)
    },

    newSource () {
      this.$router.push('/sources/new')
    }
  },

  watch: {
    domains: function () {
      this.checkLoading()
    },
    sources: function () {
      this.checkLoading()
    },
    sources_specs: function () {
      this.checkLoading()
    }
  }
}
</script>
