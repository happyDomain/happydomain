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
<div>
  <div v-if="isLoading" class="mt-5 d-flex justify-content-center align-items-center">
    <b-spinner variant="primary" label="Spinning" class="mr-3"></b-spinner> Retrieving source information...
  </div>
  <div v-if="!isLoading">
    <h2 class="mt-3 mb-3">
      {{ domain.domain }}
      <small class="text-muted">
        Source parameters
      </small>
    </h2>
    <p>
      <span class="text-primary">Name</span><br>
      <strong>{{ source._comment }}</strong>
    </p>
    <p>
      <span class="text-primary">Source type</span><br>
      <strong :title="source._srctype">{{ specs[source._srctype].name }}</strong><br>
      <span class="text-muted">{{ specs[source._srctype].description }}</span>
    </p>
    <p v-for="(spec,index) in source_specs.fields" v-bind:key="index" v-show="!spec.secret">
      <span class="text-primary">{{ spec.label }}</span><br>
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
