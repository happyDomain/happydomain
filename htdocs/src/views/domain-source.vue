<template>
<div>
  <div v-if="isLoading">
    <b-spinner variant="secondary" label="Spinning"></b-spinner> Retrieving source information...
  </div>
  <div v-if="!isLoading">
    <h3 class="text-primary">Source parameters <router-link :to="'/sources/' + source._id" class="badge badge-info">Change</router-link></h3>
    <p>
      <span class="text-secondary">Name</span><br>
      <strong>{{ source._comment }}</strong>
    </p>
    <p>
      <span class="text-secondary">Source type</span><br>
      <strong :title="source._srctype">{{ specs[source._srctype].name }}</strong><br>
      <span class="text-muted">{{ specs[source._srctype].description }}</span>
    </p>
    <p v-for="(spec,index) in source_specs.fields" v-bind:key="index" v-show="!spec.secret">
      <span class="text-secondary">{{ spec.label }}</span><br>
      <strong>{{ source.Source[spec.id] }}</strong>
    </p>
  </div>
</div>
</template>

<script>
import axios from 'axios'

export default {

  data: function () {
    return {
      source: null,
      source_specs: null,
      specs: null
    }
  },

  mounted () {
    axios
      .get('/api/source_specs')
      .then(response => {
        this.specs = response.data
      })
    if (this.domain != null) {
      this.updDomain()
    }
  },

  computed: {
    isLoading () {
      return this.source == null || this.source_specs == null || this.specs == null
    }
  },

  methods: {
    updDomain () {
      axios
        .get('/api/sources/' + this.domain.id_source)
        .then(
          (response) => {
            this.source = response.data

            axios
              .get('/api/source_specs/' + this.source._srctype)
              .then(response => (
                this.source_specs = response.data
              ))
          })
    }
  },

  props: ['domain'],

  watch: {
    domain: function (domain) {
      this.updDomain()
    }
  }
}
</script>
