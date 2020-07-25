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
  <b-list-group>
    <b-list-group-item v-if="isLoading" class="d-flex justify-content-center align-items-center">
      <b-spinner variant="primary" label="Spinning" class="mr-3" /> Retrieving usable sources...
    </b-list-group-item>
    <b-list-group-item v-for="(src, idx) in sources" :key="idx" :active="srcSelected === idx" button class="d-flex" @click="selectSource(idx)">
      <div class="align-self-center text-center" style="min-width:50px;width:50px;">
        <img :src="'/api/source_specs/' + idx + '/icon.png'" :alt="src.name" style="max-width: 100%; max-height: 2.5em; margin: -.6em .4em -.6em -.6em">
      </div>
      <div class="align-self-center" style="line-height: 1.1">
        <strong>{{ src.name }}</strong> &ndash;
        <small class="text-muted" :title="src.description">{{ src.description }}</small>
      </div>
    </b-list-group-item>
  </b-list-group>
</template>

<script>
import axios from 'axios'

export default {
  name: 'HNewSourceSelector',

  props: {
    value: {
      type: String,
      default: null
    }
  },

  data: function () {
    return {
      sources: null,
      srcSelected: null
    }
  },

  computed: {
    isLoading () {
      return this.sources == null
    }
  },

  mounted () {
    axios
      .get('/api/source_specs')
      .then(response => (this.sources = response.data))
  },

  methods: {
    selectSource (idx) {
      if (this.value !== null) {
        this.srcSelected = idx
        this.$emit('input', this.srcSelected)
      }
      this.$emit('sourceSelected', idx)
    }
  }
}
</script>
