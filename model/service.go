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
	Id           int       `json:"id"`
	Name         string    `json:"name"`
	Logo         string    `json:"logo"`
	MainResource string    `json:"main_resource"`
	ActiveVar    string    `json:"active_var"`
	Steps        []SStep   `json:"steps"`
}

func GetServices() ([]Service, error) {
	return []Service{
		Service{Id:  1, Name: "G Suite", Logo: "/img/services/gsuite.svg", MainResource: "MX"},
		Service{Id:  2, Name: "WordPress", Logo: "/img/services/wordpress.svg", MainResource: "A"},
		Service{Id:  3, Name: "Wix.com", Logo: "/img/services/wix.png", MainResource: "A"},
		Service{Id:  4, Name: "AWS", Logo: "/img/services/aws.svg", MainResource: "A"},
		Service{Id:  5, Name: "GCP", Logo: "/img/services/gcp.svg", MainResource: "A"},
		Service{Id:  6, Name: "Azure", Logo: "/img/services/azure.svg", MainResource: "A"},
		Service{Id:  7, Name: "Clef PGP", Logo: "/img/services/pgp.svg", MainResource: "OPENPGPKEY"},
		Service{Id:  8, Name: "Let's encrypt", Logo: "/img/services/letsencrypt.svg", MainResource: "TXT"},
		Service{Id:  9, Name: "Mailo", Logo: "/img/services/mailo.png", MainResource: "MX"},
		Service{Id: 10, Name: "ProtonMail", Logo: "/img/services/protonmail.svg", MainResource: "MX"},
		Service{Id: 11, Name: "Outlook.com", Logo: "/img/services/outlook.svg", MainResource: "MX"},
		Service{Id: 12, Name: "Drupal", Logo: "/img/services/drupal.svg", MainResource: "A"},
		Service{Id: 13, Name: "Dotclear", Logo: "/img/services/dotclear.png", MainResource: "A"},
		Service{Id: 14, Name: "OVH web", Logo: "/img/services/ovh.svg", MainResource: "A"},
		Service{Id: 15, Name: "OVH mail", Logo: "/img/services/ovh.svg", MainResource: "MX"},
		Service{Id: 16, Name: "Infomaniak mail", Logo: "/img/services/infomaniak.jpg", MainResource: "MX"},
	}, nil
}
