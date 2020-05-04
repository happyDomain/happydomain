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

package happydns

import ()

type StepInput struct {
	Type        string `json:"type"`
	Label       string `json:"label"`
	Placeholder string `json:"placeholder"`
	varname     string `json:"varname"`
	value       string `json:"value,omitempty"`
}

type StepEntries struct {
	EntryContent string `json:"entry_content"`
	Condition    string `json:"condition"`
}

type SStep struct {
	Title     string        `json:"title"`
	Body      string        `json:"body"`
	Inputs    []StepInput   `json:"inputs"`
	Entries   []StepEntries `json:"entries"`
	Condition string        `json:"condition"`
}

type Service struct {
	Id           int     `json:"id"`
	Name         string  `json:"name"`
	Logo         string  `json:"logo"`
	MainResource string  `json:"main_resource"`
	ActiveVar    string  `json:"active_var"`
	Steps        []SStep `json:"steps"`
}

func GetServices() ([]Service, error) {
	return []Service{
		Service{Id: 1, Name: "G Suite", Logo: "/img/services/gsuite.svg", MainResource: "MX"},
		Service{Id: 2, Name: "WordPress", Logo: "/img/services/wordpress.svg", MainResource: "A"},
		Service{Id: 3, Name: "Wix.com", Logo: "/img/services/wix.png", MainResource: "A"},
		Service{Id: 4, Name: "AWS", Logo: "/img/services/aws.svg", MainResource: "A"},
		Service{Id: 5, Name: "GCP", Logo: "/img/services/gcp.svg", MainResource: "A"},
		Service{Id: 6, Name: "Azure", Logo: "/img/services/azure.svg", MainResource: "A"},
		Service{Id: 7, Name: "Clef PGP", Logo: "/img/services/pgp.svg", MainResource: "OPENPGPKEY"},
		Service{Id: 8, Name: "Let's encrypt", Logo: "/img/services/letsencrypt.svg", MainResource: "TXT"},
		Service{Id: 9, Name: "Mailo", Logo: "/img/services/mailo.png", MainResource: "MX"},
		Service{Id: 10, Name: "ProtonMail", Logo: "/img/services/protonmail.svg", MainResource: "MX"},
		Service{Id: 11, Name: "Outlook.com", Logo: "/img/services/outlook.svg", MainResource: "MX"},
		Service{Id: 12, Name: "Drupal", Logo: "/img/services/drupal.svg", MainResource: "A"},
		Service{Id: 13, Name: "Dotclear", Logo: "/img/services/dotclear.png", MainResource: "A"},
		Service{Id: 14, Name: "OVH web", Logo: "/img/services/ovh.svg", MainResource: "A"},
		Service{Id: 15, Name: "OVH mail", Logo: "/img/services/ovh.svg", MainResource: "MX"},
		Service{Id: 16, Name: "Infomaniak mail", Logo: "/img/services/infomaniak.jpg", MainResource: "MX"},
	}, nil
}
