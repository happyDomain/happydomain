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
  <div v-if="service_specs" class="mb-2">
    <h-form-row
      v-for="(spec, index) in service_specs.fields"
      :key="index"
      v-model="value[spec.id]"
      :edit="edit"
      :index="index"
      :spec="spec"
    />
  </div>
</template>

<script>
import ServicesApi from '@/services/ServicesApi'

export default {
  name: 'HFormItemObject',

  components: {
    hFormRow: () => import('@/components/hFormRow')
  },

  props: {
    edit: {
      type: Boolean,
      default: false
    },
    index: {
      type: Number,
      required: true
    },
    spec: {
      type: Object,
      default: null
    },
    value: {
      type: Object,
      required: true
    }
  },

  data: function () {
    return {
      service_specs: null
    }
  },

  watch: {
    spec: function () {
      this.pullServiceSpecs()
    }
  },

  created () {
    if (this.spec) {
      this.pullServiceSpecs()
    }
  },

  methods: {
    pullServiceSpecs () {
      var type = this.spec.type[0] === '*' ? this.spec.type.substr(1) : this.spec.type
      ServicesApi.getServiceSpecs(type)
        .then(
          (response) => {
            this.service_specs = response.data
          }
        )
    }
  }
}
</script>
