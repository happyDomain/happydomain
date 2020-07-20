// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydns.org
//
// This software is a computer program whose purpose is to provide a modern
// interface to interact with DNS systems.
//
// This software is governed by the CeCILL license under French law and abiding
// by the rules of distribution of free software.  You can use, modify and/or
// redistribute the software under the terms of the CeCILL license as
// circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and rights to copy, modify
// and redistribute granted by the license, users are provided only with a
// limited warranty and the software's author, the holder of the economic
// rights, and the successive licensors have only limited liability.
//
// In this respect, the user's attention is drawn to the risks associated with
// loading, using, modifying and/or developing or reproducing the software by
// the user in light of its specific status of free software, that may mean
// that it is complicated to manipulate, and that also therefore means that it
// is reserved for developers and experienced professionals having in-depth
// computer knowledge. Users are therefore encouraged to load and test the
// software's suitability as regards their requirements in conditions enabling
// the security of their systems and/or data to be ensured and, more generally,
// to use and operate it in the same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL license and that you accept its terms.

import Vue from 'vue'
import App from './App.vue'
import router from './router'
import {
  AlertPlugin,
  BadgePlugin,
  BIcon,
  BIconArrowRight,
  BIconCloudDownload,
  BIconCloudUpload,
  BIconCheck,
  BIconChevronDown,
  BIconChevronLeft,
  BIconChevronRight,
  BIconChevronUp,
  BIconGridFill,
  BIconLink,
  BIconLink45deg,
  BIconListTask,
  BIconListUl,
  BIconPencil,
  BIconPerson,
  BIconPersonCheck,
  BIconPersonPlusFill,
  BIconPlus,
  BIconServer,
  BIconTrash,
  BIconTrashFill,
  BIconXCircle,
  ButtonGroupPlugin,
  ButtonPlugin,
  CardPlugin,
  DropdownPlugin,
  FormPlugin,
  FormCheckboxPlugin,
  FormGroupPlugin,
  FormInputPlugin,
  FormRadioPlugin,
  FormSelectPlugin,
  InputGroupPlugin,
  LayoutPlugin,
  ListGroupPlugin,
  ModalPlugin,
  NavbarPlugin,
  PopoverPlugin,
  SpinnerPlugin,
  TablePlugin,
  TabsPlugin,
  ToastPlugin
} from 'bootstrap-vue'

import HLogo from '@/components/logo.vue'

import './registerServiceWorker.js'

import './app.scss'

Vue.use(AlertPlugin)
Vue.use(BadgePlugin)
Vue.use(ButtonGroupPlugin)
Vue.use(ButtonPlugin)
Vue.use(CardPlugin)
Vue.use(DropdownPlugin)
Vue.use(FormPlugin)
Vue.use(FormCheckboxPlugin)
Vue.use(FormInputPlugin)
Vue.use(FormGroupPlugin)
Vue.use(FormRadioPlugin)
Vue.use(FormSelectPlugin)
Vue.use(LayoutPlugin)
Vue.use(InputGroupPlugin)
Vue.use(ListGroupPlugin)
Vue.use(ModalPlugin)
Vue.use(NavbarPlugin)
Vue.use(PopoverPlugin)
Vue.use(SpinnerPlugin)
Vue.use(TablePlugin)
Vue.use(TabsPlugin)
Vue.use(ToastPlugin)

Vue.component('BIcon', BIcon)
Vue.component('BIconArrowRight', BIconArrowRight)
Vue.component('BIconCloudDownload', BIconCloudDownload)
Vue.component('BIconCloudUpload', BIconCloudUpload)
Vue.component('BIconCheck', BIconCheck)
Vue.component('BIconChevronDown', BIconChevronDown)
Vue.component('BIconChevronLeft', BIconChevronLeft)
Vue.component('BIconChevronRight', BIconChevronRight)
Vue.component('BIconChevronUp', BIconChevronUp)
Vue.component('BIconGridFill', BIconGridFill)
Vue.component('BIconLink', BIconLink)
Vue.component('BIconLink45deg', BIconLink45deg)
Vue.component('BIconListTask', BIconListTask)
Vue.component('BIconListUl', BIconListUl)
Vue.component('BIconPencil', BIconPencil)
Vue.component('BIconPerson', BIconPerson)
Vue.component('BIconPersonCheck', BIconPersonCheck)
Vue.component('BIconPersonPlusFill', BIconPersonPlusFill)
Vue.component('BIconPlus', BIconPlus)
Vue.component('BIconServer', BIconServer)
Vue.component('BIconTrash', BIconTrash)
Vue.component('BIconTrashFill', BIconTrashFill)
Vue.component('BIconXCircle', BIconXCircle)

Vue.component('HLogo', HLogo)

Vue.config.productionTip = process.env.NODE_ENV === 'production'

new Vue({
  router,
  render: function (h) { return h(App) }
}).$mount('#app')

var tagsToReplace = {
  '&': '&amp;',
  '<': '&lt;',
  '>': '&gt;'
}
Vue.prototype.escapeHTML = function (str) {
  return str.replace(/[&<>]/g, function (tag) {
    return tagsToReplace[tag] || tag
  })
}

Vue.filter('fqdn', function (input, origin) {
  if (input[-1] === '.') {
    return input
  } else if (input === '') {
    return origin
  } else {
    return input + '.' + origin
  }
})
Vue.filter('hLabel', function (input) {
  if (input.label) {
    return input.label
  } else {
    return input.id
  }
})
Vue.filter('nsclass', function (input) {
  switch (input) {
    case 1:
      return 'IN'
    case 3:
      return 'CH'
    case 4:
      return 'HS'
    case 254:
      return 'NONE'
    default:
      return '##'
  }
})
Vue.filter('nsrrtype', function (input) {
  switch (input) {
    case 1: return 'A'
    case 2: return 'NS'
    case 3: return 'MD'
    case 4: return 'MF'
    case 5: return 'CNAME'
    case 6: return 'SOA'
    case 7: return 'MB'
    case 8: return 'MG'
    case 9: return 'MR'
    case 10: return 'NULL'
    case 11: return 'WKS'
    case 12: return 'PTR'
    case 13: return 'HINFO'
    case 14: return 'MINFO'
    case 15: return 'MX'
    case 16: return 'TXT'
    case 17: return 'RP'
    case 18: return 'AFSDB'
    case 19: return 'X25'
    case 20: return 'ISDN'
    case 21: return 'RT'
    case 22: return 'NSAP'
    case 23: return 'NSAP-PTR'
    case 24: return 'SIG'
    case 25: return 'KEY'
    case 26: return 'PX'
    case 27: return 'GPOS'
    case 28: return 'AAAA'
    case 29: return 'LOC'
    case 30: return 'NXT'
    case 31: return 'EID'
    case 32: return 'NIMLOC'
    case 33: return 'SRV'
    case 34: return 'ATMA'
    case 35: return 'NAPTR'
    case 36: return 'KX'
    case 37: return 'CERT'
    case 38: return 'A6'
    case 39: return 'DNAME'
    case 40: return 'SINK'
    case 41: return 'OPT'
    case 42: return 'APL'
    case 43: return 'DS'
    case 44: return 'SSHFP'
    case 45: return 'IPSECKEY'
    case 46: return 'RRSIG'
    case 47: return 'NSEC'
    case 48: return 'DNSKEY'
    case 49: return 'DHCID'
    case 50: return 'NSEC3'
    case 51: return 'NSEC3PARAM'
    case 52: return 'TLSA'
    case 53: return 'SMIMEA'
    case 55: return 'HIP'
    case 56: return 'NINFO'
    case 57: return 'RKEY'
    case 58: return 'TALINK'
    case 59: return 'CDS'
    case 60: return 'CDNSKEY'
    case 61: return 'OPENPGPKEY'
    case 62: return 'CSYNC'
    case 63: return 'ZONEMD'
    case 99: return 'SPF'
    case 100: return 'UINFO'
    case 101: return 'UID'
    case 102: return 'GID'
    case 103: return 'UNSPEC'
    case 104: return 'NID'
    case 105: return 'L32'
    case 106: return 'L64'
    case 107: return 'LP'
    case 108: return 'EUI48'
    case 109: return 'EUI64'
    case 249: return 'TKEY'
    case 250: return 'TSIG'
    case 251: return 'IXFR'
    case 252: return 'AXFR'
    case 253: return 'MAILB'
    case 254: return 'MAILA'
    case 256: return 'URI'
    case 257: return 'CAA'
    case 258: return 'AVC'
    case 259: return 'DOA'
    case 260: return 'AMTRELAY'
    case 32768: return 'TA'
    case 32769: return 'DLV'
    default: return '#'
  }
})
