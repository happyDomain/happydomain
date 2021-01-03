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

package config // import "happydns.org/config"

import (
	"flag"
	"fmt"

	"git.happydns.org/happydns/storage"
)

// declareFlags registers flags for the structure Options.
func (o *Options) declareFlags() {
	flag.StringVar(&o.DevProxy, "dev", o.DevProxy, "Proxify traffic to this host for static assets")
	flag.StringVar(&o.AdminBind, "admin-bind", o.AdminBind, "Bind port/socket for administration interface")
	flag.StringVar(&o.Bind, "bind", ":8081", "Bind port/socket")
	flag.StringVar(&o.ExternalURL, "externalurl", o.ExternalURL, "Begining of the URL, before the base, that should be used eg. in mails")
	flag.StringVar(&o.BaseURL, "baseurl", o.BaseURL, "URL prepended to each URL")
	flag.StringVar(&o.DefaultNameServer, "default-ns", o.DefaultNameServer, "Adress to the default name server")
	flag.Var(&o.StorageEngine, "storage-engine", fmt.Sprintf("Select the storage engine between %v", storage.GetStorageEngines()))
	flag.BoolVar(&o.NoAuth, "no-auth", false, "Disable user access control, use default account")

	// Others flags are declared in some other files likes sources, storages, ... when they need specials configurations
}

// parseCLI parse the flags and treats extra args as configuration filename.
func (o *Options) parseCLI() error {
	flag.Parse()

	for _, conf := range flag.Args() {
		err := o.parseFile(conf)
		if err != nil {
			return err
		}
	}

	return nil
}
