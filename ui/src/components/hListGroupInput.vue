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
    <form @submit.stop.prevent="$emit('submit', $event)">
      <b-list-group-item class="d-flex justify-content-between align-items-center">
        <b-input-group>
          <template #prepend>
            <b-input-group-prepend style="width: 50px" @click="$refs.lginput.focus()">
              <b-icon icon="plus" style="width: 100%; height: 2.3rem; margin-right: -.6em; margin-left: -.6em" />
            </b-input-group-prepend>
          </template>
          <b-form-input
            ref="lginput"
            :autofocus="autofocus"
            :class="inputClass"
            :placeholder="placeholder"
            :state="value.length ? state : null"
            style="border:none;box-shadow:none;z-index:0"
            :value="value"
            @input="$emit('input', $event)"
            @update="$emit('update', $event)"
          />
          <template #append>
            <b-input-group-append v-show="value.length">
              <b-button type="submit" variant="outline-primary">
                {{ $t('common.add-new-thing', { thing: $t('domains.kind') }) }}
              </b-button>
            </b-input-group-append>
          </template>
        </b-input-group>
      </b-list-group-item>
    </form>
  </b-list-group>
</template>

<script>
export default {
  name: 'HListGroupInput',

  props: {
    autofocus: {
      type: Boolean,
      default: false
    },
    inputClass: {
      type: String,
      default: ''
    },
    placeholder: {
      type: String,
      default: ''
    },
    state: {
      type: Boolean,
      default: null
    },
    value: {
      type: String,
      required: true
    }
  }
}
</script>
