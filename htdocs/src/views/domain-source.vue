<template>
  <div>
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
    <p v-for="(spec,index) in source_specs" v-bind:key="index" v-show="!spec.secret">
      <span class="text-secondary">{{ spec.label }}</span><br>
      <strong>{{ source.Source[spec.id] }}</strong>
    </p>
  </div>
</template>

<script>
import axios from 'axios'

export default {

  data: function () {
    return {
      source: {},
      source_specs: {},
      specs: {}
    }
  },

  mounted () {
    axios
      .get('/api/source_specs')
      .then(response => (
        this.specs = response.data
      ))
  },

  props: ['domain'],

  watch: {
    domain: function (domain) {
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
  }
}
</script>
