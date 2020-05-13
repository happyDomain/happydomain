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
    <b-container v-for="(val, key) in value" :key="key">
      <label :title="key" class="col-md-4 col-form-label text-truncate text-md-right text-primary">{{ key }}</label>
      <b-col md="8">
        <h-form-item v-for="(val, key) in Object.keys(value)" :key="key" v-model="value[key]" :spec="service_specs" :edit="edit" :index="index" />
      </b-col>
    </b-container>
  </div>
</template>

<script>
export default {
  name: 'HFormItemMap',

  components: {
    hFormItem: () => import('@/components/hFormItem')
  },

  props: {
    edit: Boolean,
    index: Number,
    spec: Object,
    value: {}
  },

  computed: {
    service_specs () {
      var newSpec = Object.assign({}, this.spec)
      newSpec.type = newSpec.type.substr(11)
      return newSpec
    }
  }
}
</script>
