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
  <b-container class="pb-4">
    <i18n path="common.welcome" tag="h1" class="text-center mb-4">
      <h-logo height="40" />
    </i18n>
    <b-card-group class="my-4" deck>
      <b-card>
        <h3 class="text-secondary text-center mt-1 mb-4">
          {{ $t('onboarding.no-sale.title') }}
        </h3>
        <i18n path="onboarding.no-sale.description" tag="p" class="text-justify text-indent mt-4 mb-3">
          <h-logo height="19" />
        </i18n>
        <p class="text-justify text-indent mt-3 mb-4">
          {{ $t('onboarding.no-sale.buy-advice') }}
        </p>
      </b-card>
      <b-card>
        <h3 class="text-primary text-center mt-1 mb-4">
          {{ $t('onboarding.own') }}
        </h3>
        <p class="text-justify text-indent my-4">
          <i18n path="onboarding.use" tag="span">
            <template #happyDNS>
              <h-logo height="19" />
            </template>
            <template #first-step>
              <span v-if="noSource">{{ $t('onboarding.suggest-source') }}</span><i18n path="onboarding.choose-configured" tag="span">
                <router-link to="/sources/new">
                  {{ $t('onboarding.add-one') }}
                </router-link>
              </i18n>
            </template>
          </i18n>
        </p>
        <source-list v-if="!noSource" emit-new-if-empty no-label @new-source="noSource = true" @source-selected="selectExistingSource" />
        <h-new-source-selector v-if="noSource" @source-selected="selectNewSource" />
      </b-card>
    </b-card-group>

    <b-card id="aa-hosting" class="my-3">
      <span class="text-secondary font-weight-bold">{{ $t('onboarding.questions.hosting.q') }}</span><br>
      <div class="mx-3">
        {{ $t('onboarding.questions.hosting.a') }}
      </div>
    </b-card>

    <b-card id="sec-hosting" class="my-3">
      <i18n path="onboarding.questions.secondary.q" tag="span" class="text-secondary font-weight-bold">
        <h-logo height="18" />
      </i18n>
      <br>
      <div class="mx-3">
        {{ $t('onboarding.questions.secondary.a') }}
      </div>
    </b-card>
  </b-container>
</template>

<script>
export default {
  components: {
    hNewSourceSelector: () => import('@/components/hNewSourceSelector'),
    sourceList: () => import('@/components/sourceList')
  },

  data () {
    return {
      noSource: false
    }
  },

  methods: {
    selectExistingSource (src) {
      this.$router.push('/sources/' + encodeURIComponent(src._id) + '/domains')
    },
    selectNewSource (index, src) {
      this.$router.push('/sources/new/' + encodeURIComponent(index) + '/0')
    }
  }
}
</script>
