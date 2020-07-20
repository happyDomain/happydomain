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
    <h1 class="text-center my-4">
      Welcome to <h-logo height="40" />!
    </h1>

    <b-card-group class="my-4" deck>
      <b-card>
        <h3 class="text-secondary text-center mt-1 mb-4">
          I don't own any domain
        </h3>
        <p class="text-justify text-indent mt-4 mb-3">
          <h-logo height="19" /> does not sell domain yet. To start using our interface, you need to buy a domain from one of our supported provider.
        </p>
        <p class="text-justify text-indent mt-3 mb-4">
          We'll provide some guidance in a near future on how to easily buy a domain name. So stay tune and get in touch with us if you'll help us to build a comprehensive guide.
        </p>
      </b-card>
      <b-card>
        <h3 class="text-primary text-center mt-1 mb-4">
          I already own domain(s)
        </h3>
        <p class="text-justify text-indent my-4">
          Use <h-logo height="19" /> as a remplacement interface to your usual domain name provider. It'll still rely on your provider's infrastructure, you'll just take benefit from our simple interface. As a first step, <span v-if="noSource">choose your provider:</span><span v-else>choose between already configured providers or <router-link to="/sources/new">add a new one</router-link>:</span>
        </p>
        <source-list v-if="!noSource" emit-new-if-empty no-label @newSource="noSource = true" @sourceSelected="selectExistingSource" />
        <h-new-source-selector v-if="noSource" @sourceSelected="selectNewSource" />
      </b-card>
    </b-card-group>

    <b-card id="aa-hosting" class="my-3">
      <span class="text-secondary font-weight-bold">I don't want to rely on my domain name hosting provider anymore. Can I host my domain name on your infrastructure?</span><br>
      <div class="mx-3">
        We'll provide such feature in a near future, as it's on our manifest. We choose to focus first on spreading the word that domain names are accessibles to everyone through this sweet interface, before targeting privacy, censorship, &hellip;
      </div>
    </b-card>

    <b-card id="sec-hosting" class="my-3">
      <span class="text-secondary font-weight-bold">I've my own infrastructure, can I use <h-logo height="18" /> as secondary authoritative server?</span><br>
      <div class="mx-3">
        We'll provide such feature in a near future, as soon as our name server infrastructure is on.
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
