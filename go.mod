module git.happydns.org/happydomain

go 1.16

require (
	github.com/StackExchange/dnscontrol/v3 v3.20.0
	github.com/gin-gonic/gin v1.7.7
	github.com/go-mail/mail v2.3.1+incompatible
	github.com/go-sql-driver/mysql v1.6.0
	github.com/golang-jwt/jwt/v4 v4.4.2
	github.com/miekg/dns v1.1.50
	github.com/ovh/go-ovh v1.1.0
	github.com/syndtr/goleveldb v1.0.0
	github.com/yuin/goldmark v1.4.14
	golang.org/x/crypto v0.0.0-20220829220503-c86fa9a7ed90
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/mail.v2 v2.3.1 // indirect
)

replace github.com/StackExchange/dnscontrol/v3 => github.com/nemunaire/dnscontrol/v3 v3.20.10
