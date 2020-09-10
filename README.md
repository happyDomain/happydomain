happyDNS
========

Finally a simple, modern and open source interface for domain name.

It consists of a HTTP REST API written in Golang (primarily based on https://github.com/miekg/dns) with a nice web interface written in Vue.js.
It runs as a single stateless Linux binary, backed by a database (currently: LevelDB, more SGBD to come soon).

Features
--------

TODO

Building
--------

### Dependencies

In order to build the happyDNS project, you'll need the following dependencies:

* `go` at least version 1.13
* `go-bindata`
* `nodejs` tested with version 14.4.0
* `yarn` tested with version 1.22.4


### Instructions

1. First, I'll need to prepare the frontend.

Go inside the `htdocs/` directory and install the node modules dependencies:

```
cd htdocs/
yarn install
```

2. Generates assets files used by Go code:

```
cd .. # Go back to the root of the project
go generate
```

3. Build the Go code:

```
go build
```

The command will create a binary `happydns` you can use standalone.


Install at home
---------------

The binary comes with sane default options to start with.
You can simply launch the following command in your terminal:

```
./happydns
```

After some initialization, it should show you:

    Admin listening on ./happydns.sock
    Ready, listening on :8081

Go to http://localhost:8081/ to start using happyDNS.


### Database configuration

By default, the LevelDB storage engine is used. You can change the storage engine using the option `-storage-engine other-engine`.

The help command `./happydns -help` can show you the available engines. By example:

    -storage-engine value
    	Select the storage engine between [leveldb mysql] (default leveldb)

#### LevelDB

    -leveldb-path string
    	Path to the LevelDB Database (default "happydns.db")

By default, a new directory is created near the binary, called `happydns.db`. This directory contains the database used by the program. You can change it to a more


### Persistant configuration

The binary will automatically look for some existing configuration files:

* `./happydns.conf` in the current directory;
* `$XDG_CONFIG_HOME/happydns/happydns.conf`;
* `/etc/happydns.conf`.

Only the first file found will be used.

It is also possible to specify a custom path by adding it as argument to the command line:

```
./happydns /etc/happydns/config
```

#### Config file format

Comments line has to begin with #, it is not possible to have comments at the end of a line, by appending # followed by a comment.

Place on each line the name of the config option and the expected value, separated by `=`. For example:

```
storage-engine=leveldb
leveldb-path=/var/lib/happydns/db/
```

#### Environment variables

It'll also look for special environment variables, beginning with `HAPPYDNS_`.

You can achieve the same as the previous example, with the following environment variables:

```
HAPPYDNS_STORAGE_ENGINE=leveldb
HAPPYDNS_LEVELDB_PATH=/var/lib/happydns/db/
```

You just have to replace dash by underscore.


Development environment
-----------------------

If you want to contribute to the frontend, instead of regenerating the frontend assets each time you made a modification (with `go generate`), you can use the development tools:

In one terminal, run `happydns` with the following arguments:

```
./happydns -dev http://127.0.0.1:8080
```

In another terminal, run the node part:

```
cd htdocs/
yarn run serve
```

With this setup, static assets integrated inside the go binary will not be used, instead it'll forward all request for static assets to the node server, that do dynamic reload, etc.
