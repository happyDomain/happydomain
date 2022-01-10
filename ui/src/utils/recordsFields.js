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

export function recordsFields (rrtype) {
  switch (rrtype) {
    case 1:
      return ['A']
    case 2:
      return ['Ns']
    case 5:
      return ['Target']
    case 6:
      return ['Ns', 'Mbox', 'Serial', 'Refresh', 'Retry', 'Expire', 'Minttl']
    case 12:
      return ['Ptr']
    case 13:
      return ['Cpu', 'Os']
    case 15:
      return ['Mx', 'Preference']
    case 16:
    case 99:
      return ['Txt']
    case 28:
      return ['AAAA']
    case 33:
      return ['Target', 'Port', 'Priority', 'Weight']
    case 43:
      return ['KeyTag', 'Algorithm', 'DigestType', 'Digest']
    case 44:
      return ['Algorithm', 'Type', 'FingerPrint']
    case 46:
      return ['TypeCovered', 'Algorithm', 'Labels', 'OrigTtl', 'Expiration', 'Inception', 'KeyTag', 'SignerName', 'Signature']
    case 52:
      return ['Usage', 'Selector', 'MatchingType', 'Certificate']
    default:
      console.warn('Unknown RRtype asked fields: ', rrtype)
      return []
  }
}
