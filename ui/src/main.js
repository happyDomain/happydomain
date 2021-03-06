// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydomain.org
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
  BIconChatRightText,
  BIconCheck,
  BIconChevronDown,
  BIconChevronLeft,
  BIconChevronRight,
  BIconChevronUp,
  BIconExclamationCircle,
  BIconExclamationOctagon,
  BIconExclamationTriangle,
  BIconGridFill,
  BIconLink,
  BIconLink45deg,
  BIconList,
  BIconListTask,
  BIconListUl,
  BIconMenuButtonWideFill,
  BIconPencil,
  BIconPerson,
  BIconPersonCheck,
  BIconPersonPlusFill,
  BIconPlus,
  BIconQuestionCircleFill,
  BIconServer,
  BIconTrash,
  BIconTrashFill,
  BIconThreeDots,
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
import i18n from './i18n'
import store from './store'

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
Vue.component('BIconChatRightText', BIconChatRightText)
Vue.component('BIconCheck', BIconCheck)
Vue.component('BIconChevronDown', BIconChevronDown)
Vue.component('BIconChevronLeft', BIconChevronLeft)
Vue.component('BIconChevronRight', BIconChevronRight)
Vue.component('BIconChevronUp', BIconChevronUp)
Vue.component('BIconExclamationCircle', BIconExclamationCircle)
Vue.component('BIconExclamationOctagon', BIconExclamationOctagon)
Vue.component('BIconExclamationTriangle', BIconExclamationTriangle)
Vue.component('BIconGridFill', BIconGridFill)
Vue.component('BIconLink', BIconLink)
Vue.component('BIconLink45deg', BIconLink45deg)
Vue.component('BIconList', BIconList)
Vue.component('BIconListTask', BIconListTask)
Vue.component('BIconListUl', BIconListUl)
Vue.component('BIconMenuButtonWideFill', BIconMenuButtonWideFill)
Vue.component('BIconPencil', BIconPencil)
Vue.component('BIconPerson', BIconPerson)
Vue.component('BIconPersonCheck', BIconPersonCheck)
Vue.component('BIconPersonPlusFill', BIconPersonPlusFill)
Vue.component('BIconPlus', BIconPlus)
Vue.component('BIconQuestionCircleFill', BIconQuestionCircleFill)
Vue.component('BIconServer', BIconServer)
Vue.component('BIconTrash', BIconTrash)
Vue.component('BIconTrashFill', BIconTrashFill)
Vue.component('BIconThreeDots', BIconThreeDots)
Vue.component('BIconXCircle', BIconXCircle)

Vue.component('HLogo', HLogo)

Vue.config.productionTip = process.env.NODE_ENV === 'production'

new Vue({
  store,
  router,
  i18n,
  render: h => h(App)
}).$mount('#app')

const tagsToReplace = {
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
Vue.filter('nsttl', function (input) {
  input = Number(input)

  let ret = ''
  if (input / 86400 >= 1) {
    ret = Math.floor(input / 86400) + 'd '
    input = input % 86400
  }
  if (input / 3600 >= 1) {
    ret = Math.floor(input / 3600) + 'h '
    input = input % 3600
  }
  if (input / 60 >= 1) {
    ret = Math.floor(input / 60) + 'm '
    input = input % 60
  }
  if (input >= 1) {
    ret = Math.floor(input) + 's'
  }

  return ret
})
Vue.filter('nsrrtype', function (input) {
  switch (input) {
    case '1': case 1: return 'A'
    case '2': case 2: return 'NS'
    case '3': case 3: return 'MD'
    case '4': case 4: return 'MF'
    case '5': case 5: return 'CNAME'
    case '6': case 6: return 'SOA'
    case '7': case 7: return 'MB'
    case '8': case 8: return 'MG'
    case '9': case 9: return 'MR'
    case '10': case 10: return 'NULL'
    case '11': case 11: return 'WKS'
    case '12': case 12: return 'PTR'
    case '13': case 13: return 'HINFO'
    case '14': case 14: return 'MINFO'
    case '15': case 15: return 'MX'
    case '16': case 16: return 'TXT'
    case '17': case 17: return 'RP'
    case '18': case 18: return 'AFSDB'
    case '19': case 19: return 'X25'
    case '20': case 20: return 'ISDN'
    case '21': case 21: return 'RT'
    case '22': case 22: return 'NSAP'
    case '23': case 23: return 'NSAP-PTR'
    case '24': case 24: return 'SIG'
    case '25': case 25: return 'KEY'
    case '26': case 26: return 'PX'
    case '27': case 27: return 'GPOS'
    case '28': case 28: return 'AAAA'
    case '29': case 29: return 'LOC'
    case '30': case 30: return 'NXT'
    case '31': case 31: return 'EID'
    case '32': case 32: return 'NIMLOC'
    case '33': case 33: return 'SRV'
    case '34': case 34: return 'ATMA'
    case '35': case 35: return 'NAPTR'
    case '36': case 36: return 'KX'
    case '37': case 37: return 'CERT'
    case '38': case 38: return 'A6'
    case '39': case 39: return 'DNAME'
    case '40': case 40: return 'SINK'
    case '41': case 41: return 'OPT'
    case '42': case 42: return 'APL'
    case '43': case 43: return 'DS'
    case '44': case 44: return 'SSHFP'
    case '45': case 45: return 'IPSECKEY'
    case '46': case 46: return 'RRSIG'
    case '47': case 47: return 'NSEC'
    case '48': case 48: return 'DNSKEY'
    case '49': case 49: return 'DHCID'
    case '50': case 50: return 'NSEC3'
    case '51': case 51: return 'NSEC3PARAM'
    case '52': case 52: return 'TLSA'
    case '53': case 53: return 'SMIMEA'
    case '55': case 55: return 'HIP'
    case '56': case 56: return 'NINFO'
    case '57': case 57: return 'RKEY'
    case '58': case 58: return 'TALINK'
    case '59': case 59: return 'CDS'
    case '60': case 60: return 'CDNSKEY'
    case '61': case 61: return 'OPENPGPKEY'
    case '62': case 62: return 'CSYNC'
    case '63': case 63: return 'ZONEMD'
    case '99': case 99: return 'SPF'
    case '100': case 100: return 'UINFO'
    case '101': case 101: return 'UID'
    case '102': case 102: return 'GID'
    case '103': case 103: return 'UNSPEC'
    case '104': case 104: return 'NID'
    case '105': case 105: return 'L32'
    case '106': case 106: return 'L64'
    case '107': case 107: return 'LP'
    case '108': case 108: return 'EUI48'
    case '109': case 109: return 'EUI64'
    case '249': case 249: return 'TKEY'
    case '250': case 250: return 'TSIG'
    case '251': case 251: return 'IXFR'
    case '252': case 252: return 'AXFR'
    case '253': case 253: return 'MAILB'
    case '254': case 254: return 'MAILA'
    case '256': case 256: return 'URI'
    case '257': case 257: return 'CAA'
    case '258': case 258: return 'AVC'
    case '259': case 259: return 'DOA'
    case '260': case 260: return 'AMTRELAY'
    case '32768': case 32768: return 'TA'
    case '32769': case 32769: return 'DLV'
    default: return '#'
  }
})
