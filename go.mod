module git.happydns.org/happydns

go 1.16

require (
	github.com/StackExchange/dnscontrol/v3 v3.13.0
	github.com/gin-gonic/gin v1.7.7
	github.com/go-mail/mail v2.3.1+incompatible
	github.com/go-playground/validator/v10 v10.7.0 // indirect
	github.com/go-sql-driver/mysql v1.6.0
	github.com/hashicorp/go-retryablehttp v0.7.0 // indirect
	github.com/json-iterator/go v1.1.11 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/miekg/dns v1.1.43
	github.com/ovh/go-ovh v1.1.0
	github.com/syndtr/goleveldb v1.0.0
	github.com/ugorji/go v1.2.6 // indirect
	github.com/yuin/goldmark v1.4.4
	golang.org/x/crypto v0.0.0-20211209193657-4570a0811e8b
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/mail.v2 v2.3.1 // indirect
	gopkg.in/ns1/ns1-go.v2 v2.6.0 // indirect
)

replace github.com/StackExchange/dnscontrol/v3 => github.com/nemunaire/dnscontrol/v3 v3.13.20
