package happydns

import (
	"encoding/base64"
)

type Zone struct {
	Id              int64 `json:"id"`
	IdUser          int64
	DomainName      string `json:"domain"`
	Server          string `json:"server,omitempty"`
	KeyName         string `json:"keyname,omitempty"`
	KeyAlgo         string `json:"algorithm,omitempty"`
	KeyBlob         []byte `json:"keyblob,omitempty"`
	StorageFacility string `json:"storage_facility,omitempty"`
}

type Zones []*Zone

func NewZone(u User, dn, server, keyname, algo string, keyblob []byte, storage string) *Zone {
	return &Zone{
		IdUser:          u.Id,
		DomainName:      dn,
		Server:          server,
		KeyName:         keyname,
		KeyAlgo:         algo,
		KeyBlob:         keyblob,
		StorageFacility: storage,
	}
}

func (z *Zone) Base64KeyBlob() string {
	return base64.StdEncoding.EncodeToString(z.KeyBlob)
}
