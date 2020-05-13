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
  <b-form-row
    v-show="edit || value"
  >
    <label v-if="spec.label" :for="'spec-' + index" :title="spec.label" class="col-md-4 col-form-label text-truncate text-md-right text-primary">{{ spec.label }}</label>
    <label v-else :for="'spec-' + index" :title="spec.label" class="col-md-4 col-form-label text-truncate text-md-right text-primary">{{ spec.id }}</label>
    <b-col md="8">
      <h-form-item v-model="value" :spec="spec" :edit="edit" :index="index" />
      <p v-if="spec.description" class="text-justify" style="line-height: 1.1">
        <small class="text-muted">{{ spec.description }}</small>
      </p>
    </b-col>
  </b-form-row>
</template>

<script>
export default {
  name: 'HFormRow',
  components: {
    hFormItem: () => import('@/components/hFormItem')
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
    // eslint-disable-next-line
    value: {
      required: true
    }
  }
}
</script>
