package config // import "happydns.org/config"

import (
	"flag"
)

func (o *Options) parseCLI() error {
	flag.StringVar(&o.DevProxy, "dev", o.DevProxy, "Proxify traffic to this host for static assets")
	flag.StringVar(&o.Bind, "bind", ":8081", "Bind port/socket")
	flag.StringVar(&o.DSN, "dsn", o.DSN, "DSN to connect to the MySQL server")
	flag.StringVar(&o.BaseURL, "baseurl", o.BaseURL, "URL prepended to each URL")
	flag.StringVar(&o.DefaultNameServer, "defaultns", o.DefaultNameServer, "Adress to the default name server")
	flag.Parse()

	for _, conf := range flag.Args() {
		err := o.parseFile(conf)
		if err != nil {
			return err
		}
	}

	return nil
}
