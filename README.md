happyDomain
===========
[Website](https://www.happydomain.org/) · [Try it online](https://try.happydomain.org/) · [Documentation](https://help.happydomain.org/) · [Chat on Matrix](https://matrix.to/#/%23happyDNS:matrix.org) · [Share feedback](https://feedback.happydomain.org/)

Summary
-------

happyDomain is a self-hostable, open-source web interface that unifies DNS management across 60+ registrars and providers, reviews every change as a diff before it goes live, and runs 35+ automated checks to catch DNS problems before they cause outages.

![Screenshots of happyDomain](./docs/header.png)


Table of Contents
-----------------

- [What is happyDomain?](#what-is-happydomain)
- [Who's it for?](#whos-it-for)
- [Features](#features)
- [Getting started](#getting-started)
  - [With Docker](#with-docker)
  - [From binary](#from-binary)
- [Configuration](#configuration)
  - [Storage engine](#storage-engine)
  - [Persistent configuration](#persistent-configuration)
  - [Network and deployment notes](#network-and-deployment-notes)
- [Building and development](#building-and-development)
  - [Building from source](#building-from-source)
  - [Development environment](#development-environment)
- [Contributing](#contributing)
- [AI Disclaimer](#ai-disclaimer)
- [License](#license)


What is happyDomain?
---------------------

Most DNS tooling is either your registrar's web console (one interface per provider, no history, no review step) or hand-edited zone files: powerful, but unforgiving, since one bad record can take a service down.

happyDomain sits in between. It talks to your registrars and DNS providers through a common interface, so managing five domains across three providers feels like managing one. Related records are grouped into logical services, such as a domain's email setup, its website, or a VPN, instead of a flat list of resource records: you're editing intent, not raw DNS syntax. Every change goes through a diff view before it's published, with a full history and rollback if something goes wrong. In the background, a library of 35+ checks continuously watches your zones for the kind of mistakes that cause real outages: DNSSEC misconfiguration, dangling records pointing at services you no longer control, expiring domains, broken mail authentication.

It's built on Go and [DNSControl](https://dnscontrol.org/), runs as a single stateless binary, and is fully self-hostable under the AGPL. Or skip installing anything and [try it online](https://try.happydomain.org/) first.


Who's it for?
-------------

- **Sysadmins and IT teams** juggling more domains than their registrar's dashboard was built for, who want one place to see what changed and why.
- **Freelancers and agencies** managing DNS for clients, where a wrong record is someone else's outage: the diff-before-publish workflow and audit trail matter here.
- **DevOps teams** who already review infrastructure changes like code and want DNS held to the same standard, scripted through the REST API if needed.
- **Self-hosters** running their own BIND, Knot, or PowerDNS servers who'd rather click through a real interface than hand-edit zone files.


Features
--------

|                               |                                                                                                                                                                                     |
| ----------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Multi-provider**            | 60+ registrars and DNS providers via [DNSControl](https://dnscontrol.org/), plus self-hosted BIND, Knot, and PowerDNS over RFC 2136                                                 |
| **Services, not raw records** | Related records (an email setup, a website, a VPN) are grouped into logical services instead of a flat zone file                                                                    |
| **Review before you publish** | A diff view shows exactly what will change before it goes live, with a full history and rollback of everything published                                                            |
| **35+ automated checks**      | DNSSEC, dangling records, TLS certificates, domain expiry, blacklists, and more, running continuously. Extensible via a [plugin SDK](https://github.com/happyDomain/checker-sdk-go) |
| **Import, export, automate**  | Zone import/export, and a REST API for scripting changes                                                                                                                            |
| **Bring your own auth**       | Multiple users with authentication, a single user without, or external auth via OpenID Connect / JWT (Auth0, ...)                                                                   |

> [!IMPORTANT]
> **happyDomain is functional but still very much a work in progress:** 
> 
> It's a carefully crafted proof of concept is evolving over time. Given the diversity of DNS configurations and user needs, we haven't yet identified all the bugs. If something doesn't work [tell us what's wrong](https://github.com/happyDomain/happydomain/issues). And [your feedback](https://feedback.happydomain.org/) guides where the project goes next, so please send through any and all feedback.


Quickstart guide
----------------

Two ways to run happyDomain: with Docker (fastest), or from a prebuilt binary. Pick one. [Configuration](#configuration) below covers storage, network, and deployment options once you're up and running.

### With Docker

happyDomain is a Docker-sponsored open source project, so you can try or deploy it with Docker, Podman, Kubernetes, and similar tools.

1. Run:

   ```bash
   docker run -e HAPPYDOMAIN_NO_AUTH=1 -p 8081:8081 happydomain/happydomain
   ```

   (Docker will pull the image automatically the first time — expect a short wait depending on your connection.) This launches happyDomain in a few seconds for evaluation purposes (no authentication, volatile storage, ...).

2. Open <http://localhost:8081>

#### Or, if you prefer `docker compose`:

1. Clone the repository:

   ```bash
   git clone https://framagit.org/happyDomain/happyDomain.git
   cd happyDomain
   ```

> [!TIP]
> Can't reach framagit.org (some VPNs and networks block it)? Clone the [GitHub mirror](https://github.com/happyDomain/happydomain) instead: `git clone https://github.com/happyDomain/happydomain.git happyDomain`

2. Run:

   ```bash
   docker compose up
   ```

   > [!TIP]
   > If this fails and you're on an Apple Silicon Mac, run this once, then try `docker compose up` again:
   >
   > ```bash
   > cat > docker-compose.override.yml << 'EOF'
   > services:
   >   matrixfederationtester:
   >     platform: linux/amd64
   > EOF
   > ```
   >
   > (Works around one of the optional checker services not having a version built for that chip. The error looks like `no matching manifest for linux/arm64/v8`.)

3. Open <http://localhost:8081>

> [!NOTE]
> To deploy happyDomain for proper use (not just evaluation), check the [Docker image documentation](https://hub.docker.com/r/happydomain/happydomain).

### From binary

1. Download the binary matching your operating system and CPU architecture from <https://get.happydomain.org/> (choose the latest version available, or `master`).

2. Launch it from your terminal:

   ```bash
   ./happyDomain
   ```

   After some initialization, it should show you:

       Admin listening on ./happydomain.sock
       Ready, listening on :8081

3. Go to <http://localhost:8081/> to start using happyDomain.

Configuration
-------------

What's below covers what's specific to self-hosting happyDomain: storage backends and network/deployment quirks. For every other setting (network binding, mail/SMTP, authentication, page branding, and more), see the [configuration reference](https://help.happydomain.org/en/introduction/deploy/config/).

### Storage engine

By default, happyDomain uses the LevelDB storage engine. Change it with the `-storage-engine` option; `./happyDomain -help` lists what's available:

```
    -storage-engine value
    	Select the storage engine between [inmemory leveldb oracle-nosql postgresql] (default leveldb)
```

Here's what each one means:

**LevelDB** (the default) is a small embedded key-value store, like SQLite: no extra daemon required. A new directory called `happydomain.db` is created near the binary; point it somewhere more permanent with `-leveldb-path`.

 **In-memory** stores data in memory and loses it when the service stops. Useful for quick evaluation, nothing else.

<details>
<summary><strong>PostgreSQL</strong> (for teams that already run Postgres and want to reuse it)</summary>
<p>happyDomain stores everything in a single table with <code>key</code> and <code>value</code> columns: a plain key-value scheme, not what PostgreSQL is optimized for. If you're starting from scratch and need more scale than LevelDB, a storage backend built for key-value workloads will serve you better.</p>
<pre><code>-postgres-database string
      PostgreSQL database name (default "happydomain")
-postgres-host string
      PostgreSQL server hostname (default "localhost")
-postgres-password string
      PostgreSQL password
-postgres-port int
      PostgreSQL server port (default 5432)
-postgres-ssl-mode string
      PostgreSQL SSL mode (disable, require, verify-ca, verify-full) (default "disable")
-postgres-table string
      PostgreSQL table name for key-value storage (default "happydomain_kv")
-postgres-user string
      PostgreSQL username (default "happydomain")
</code></pre>
</details>

<details>
<summary><strong>Oracle NoSQL Database</strong> (for OCI-based deployments)</summary>
<p>Oracle NoSQL Database is Oracle Cloud Infrastructure's (OCI) managed NoSQL service. To use it, create an OCI NoSQL table with a <code>key</code> field (string, primary key) and a <code>value</code> field (JSON), and authenticate with an OCI IAM API signing key. Then configure happyDomain with:</p>
<pre><code>-oci-compartment string
      OCI compartment ID where the NoSQL database lies
-oci-fingerprint string
      OCI user API key fingerprint
-oci-private-key-file string
      Path to the OCI private key for the given user
-oci-region string
      OCI region where the NoSQL database is located (default "us-phoenix-1")
-oci-table string
      Table name where values are stored (default "happydomain")
-oci-tenancy string
      OCI tenancy ID where is located the NoSQL database
-oci-user string
      OCI user ID accessing the NoSQL database
</code></pre>
</details>

### Persistent configuration

Configure happyDomain via a config file (`./happydomain.conf`, `$XDG_CONFIG_HOME/happydomain/happydomain.conf`, or `/etc/happydomain.conf`, in that order, or pass a custom path as an argument), environment variables prefixed `HAPPYDOMAIN_`, or command-line flags. 

> [!NOTE]
> See the [configuration reference](https://help.happydomain.org/en/introduction/deploy/config/) for the full option list and examples of each.

### Network and deployment notes

> [!NOTE]
> - **Behind a reverse proxy?** happyDomain ignores `X-Forwarded-For` unless you declare which proxies may set it, with `-trusted-proxy`. See [docs/reverse-proxy.md](docs/reverse-proxy.md).
> - **Managing a DNS server or router on your own network?** happyDomain only connects to publicly routable addresses by default. To manage a PowerDNS, BIND, AdGuard Home, OpenWrt, Mikrotik, or UniFi instance you host yourself, name its address with `-outbound-allowed-target`. See [docs/outbound-targets.md](docs/outbound-targets.md).
> - **Need the OVH API?** OVH doesn't issue simple API keys. It relies on a web flow, and your happyDomain instance needs a dedicated Application Key to initiate it. [Follow these instructions to connect to OVH](https://help.happydomain.org/en/introduction/deploy/ovh).

Building and development
-------------------------

### Building from source

#### Prerequisites

Before building happyDomain, install:

- [Go](https://go.dev/dl/), version 1.26 or newer
- [Node.js](https://nodejs.org/) 24
- [swag](https://github.com/swaggo/swag) 1.16:

  ```bash
  go install github.com/swaggo/swag/cmd/swag@latest
  ```

#### Instructions

1. Clone the repository (skip if you already have it from earlier):

   ```bash
   git clone https://framagit.org/happyDomain/happyDomain.git
   cd happyDomain
   ```

   > [!TIP]
   > Can't reach framagit.org (some VPNs and networks block it)? Clone the [GitHub mirror](https://github.com/happyDomain/happydomain) instead: `git clone https://github.com/happyDomain/happydomain.git happyDomain`

2. Install the frontend's node module dependencies:

   ```bash
   pushd web; npm install; popd
   pushd web-admin; npm install; popd
   ```

3. Generate Go code (API docs, icons, etc). This also produces the OpenAPI specs the frontend needs in the next step:

   ```bash
   go generate -tags swagger,web ./...
   ```

   > [!TIP]
   > Getting `exec: "swag": executable file not found in $PATH`? Run this:
   >
   > ```bash
   > echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
   > ```
   >
   > Then open a new terminal, `cd` back into the `happyDomain` folder, and run step 3 again. 
   > 
   > (Using bash instead of zsh? Use `~/.bashrc`.)

4. Build the frontend assets:

   ```bash
   go generate web/generate_npm.go
   go generate web-admin/generate_npm.go
   ```

5. Finally, build the Go code:

   ```bash
   go build -tags swagger,web ./cmd/happyDomain
   ```

This last command will create a binary `happyDomain` you can use standalone. Run it the same way as in [From binary](#from-binary).

### Development environment

If you want to contribute to the frontend, instead of regenerating the frontend assets every time you make a change (with `go generate`), you can use the development tools:

In one terminal, run `happydomain` with the following arguments:

```bash
./happyDomain -dev http://127.0.0.1:5173
```

In another terminal, run the node part:

```bash
cd web; npm run dev
```

With both running, go to <http://localhost:8081>, the same address as usual, not the Vite dev server's own port. Static assets built into the Go binary aren't used; instead, requests for them are forwarded to the node server, which reloads on every change.


Contributing
------------

Contributions are welcome! Here's how you can help:

- **Report bugs:** Open an issue on your favorite forge: [GitHub](https://github.com/happyDomain/happydomain/issues), [GitLab](https://gitlab.com/happyDomain/happydomain/-/issues), [Framagit](https://framagit.org/happyDomain/happydomain/-/issues), [Codeberg](https://codeberg.org/happyDomain/happyDomain/issues). We're highly responsive.
- **Share feedback:** [Tell us what you think](https://feedback.happydomain.org/); your input guides the project.


AI Disclaimer
-------------

There have been questions about AI usage in project development. Our project handles domain name management, a sensitive area where mistakes can cause real outages, it's important to explain how AI is used in the development process.

AI is used as a helper for:

- verification of code quality and searching for vulnerabilities
- cleaning up and improving documentation, comments and code
- assistance during development
- double-checking PRs and commits after human review

AI is not used for:

- writing entire features or components
- "vibe coding" approach
- code without line-by-line verification by a human
- code without tests

The project has:

- CI/CD pipeline automation with tests and linting to ensure code quality
- verification by experienced developers

So AI is just an assistant and a tool for developers to increase productivity and ensure code quality. The work is done by developers.

We do not differentiate between bad human code and AI vibe code. There are strict requirements for any code to be merged to keep the codebase maintainable. Even if code is written manually by a human, it's not guaranteed to be merged. Vibe code is not allowed and such PRs are rejected.

*Inspired by the [Databasus AI disclaimer](https://github.com/databasus/databasus#ai-disclaimer).*


License
-------

happyDomain is licensed under the [GNU Affero General Public License v3.0](./LICENSE) (AGPL-3.0).

A commercial license is also available, contact us if interested.
