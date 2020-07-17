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
  <div v-if="!isLoading && mySources && mySources.length > 0" class="d-flex flex-row justify-content-around flex-wrap align-self-center">
    <div v-for="(src, index) in mySources" :key="index" type="button" class="p-3 source" @click="$emit('sourceSelected', src, index)">
      <img :src="'/api/source_specs/' + src._srctype + '.png'" :alt="altNames[src['_srctype']].name">
      {{ src._comment }}
    </div>
  </div>
</template>

<script>
import axios from 'axios'

export default {
  name: 'HUserSourceSelector',

  props: {
    sources: {
      type: Object,
      default: null
    }
  },

  data: function () {
    return {
      mySources: null
    }
  },

  computed: {
    altNames () {
      if (this.sources != null) {
        return this.sources
      } else {
        var ret = {}

        for (const idx in this.mySources) {
          ret[this.mySources[idx]._srctype] = { name: idx }
        }

        return ret
      }
    },

    isLoading () {
      return this.mySources == null
    }
  },

  mounted () {
    axios
      .get('/api/sources')
      .then(response => (this.mySources = response.data))
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
    margin: 2.5% 0;
    width: 30%;
    max-width: 200px;
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
