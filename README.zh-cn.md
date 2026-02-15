happyDomain
===========

> 中文译者：[Exyone](https://www.exyone.me/)

由于软件架构限制，当前 happyDomain 仅支持简体中文，繁体中文用户请安装浏览器拓展插件以进行简繁转换。造成困扰，敬请谅解。

happyDomain 是一款免费的 Web 应用，可集中管理来自不同注册商和托管商的域名。

![happyDomain 截图](./docs/header.webp)

它由 Golang 编写的 HTTP REST API（主要基于 https://stackexchange.github.io/dnscontrol/ 和 https://github.com/miekg/dns）与 [Svelte](https://svelte.dev/) 构建的精美 Web 界面组成。
作为单一无状态的 Linux 二进制文件运行，支持多种数据库（当前支持 LevelDB，更多选项即将推出）。

**主要特性：**

* 高性能 Web 界面，响应迅速
* 支持多域名管理
* 支持 44+ DNS 提供商（含动态 DNS、RFC 2136），得益于 [DNSControl](https://stackexchange.github.io/dnscontrol/)
* 支持最新资源记录类型，得益于 [CoreDNS 库](https://github.com/miekg/dns)
* 区域编辑器支持差异对比，部署前轻松审查变更
* 保留部署变更历史记录
* 上下文帮助
* 支持多用户认证或单用户无认证模式
* 兼容外部认证（OpenId Connect 或 JWT 令牌：Auth0 等）

**happyDomain 已可投入使用，但仍需不断完善：这是一个精心打造的概念验证版本，您的反馈将助力其不断进化！**

鉴于 DNS 配置和用户需求的多样性，我们尚未发现所有潜在问题。**若遇问题，请勿离去：[向我们反馈问题所在](https://github.com/happyDomain/happydomain/issues)。** 我们响应迅速，每个报告的 bug 都能帮助改进工具，惠及众人。

[无论使用体验如何，我们都期待您的反馈！](https://feedback.happydomain.org/) 您如何看待我们简化域名管理的方式？您的初步印象有助于我们根据**您的实际期望**来指引项目方向。


使用 Docker
------------

我们是由 Docker 赞助的开源项目！因此您可以轻松使用 Docker/podman/kubernetes/... 来试用或部署应用。

使用 `docker compose` 启动 happyDomain：

```bash
git clone https://framagit.org/happyDomain/happyDomain.git
cd happyDomain
docker compose up
```

或直接使用 `docker run`：

```bash
docker run -e HAPPYDOMAIN_NO_AUTH=1 -p 8081:8081 happydomain/happydomain
```

此命令将在数秒内启动 happyDomain，用于评估测试（无认证、临时存储等）。使用浏览器访问 <http://localhost:8081> 即可体验！

部署 happyDomain，请查阅 [Docker 镜像文档](https://hub.docker.com/r/happydomain/happydomain)。


从二进制文件安装
-------------------

预编译二进制文件下载地址：<https://get.happydomain.org/>

选择目录（最新版本或 master 分支），然后选择与您的操作系统和 CPU 架构对应的二进制文件。


使用 happyDomain
---------------

二进制文件附带默认配置，可直接启动。在终端中运行以下命令即可：

```bash
./happyDomain
```

初始化完成后，应显示以下信息：

    Admin listening on ./happydomain.sock
    Ready, listening on :8081

访问 http://localhost:8081/ 开始使用 happyDomain。


### 数据库配置

默认使用 LevelDB 存储引擎。可使用 `-storage-engine` 选项更改存储引擎。

运行 `./happyDomain -help` 查看可用存储引擎：

```
    -storage-engine value
    	在 [inmemory leveldb oracle-nosql postgresql] 中选择存储引擎 (默认 leveldb)
```

#### LevelDB

LevelDB 是轻量级嵌入式键值存储（类似 SQLite，无需额外守护进程）。

```
    -leveldb-path string
    	LevelDB 数据库路径 (默认 "happydomain.db")
```

默认在二进制文件所在目录创建 `happydomain.db` 目录。可更改为更有意义或更持久的路径。

#### inmemory

数据存储于内存中，服务停止后数据即丢失。

#### PostgreSQL

PostgreSQL 支持主要面向已部署 PostgreSQL 数据库基础设施的环境。这允许您利用现有数据库设置、备份流程和运维工具，无需部署额外数据库系统。

happyDomain 以键值存储模式使用 PostgreSQL，将所有数据存储在包含 `key` 和 `value` 列的单张表中。虽然可行，但请注意，与专用键值存储相比，PostgreSQL 并非键值工作负载的最佳选择。若从头部署且需超出 LevelDB 的可扩展性，请考虑使用专为键值操作设计的存储后端。

```
    -postgres-database string
      	PostgreSQL 数据库名称 (默认 "happydomain")
    -postgres-host string
      	PostgreSQL 服务器主机名 (默认 "localhost")
    -postgres-password string
      	PostgreSQL 密码
    -postgres-port int
      	PostgreSQL 服务器端口 (默认 5432)
    -postgres-ssl-mode string
      	PostgreSQL SSL 模式 (disable, require, verify-ca, verify-full) (默认 "disable")
    -postgres-table string
      	键值存储的 PostgreSQL 表名 (默认 "happydomain_kv")
    -postgres-user string
    	PostgreSQL 用户名 (默认 "happydomain")
```

#### Oracle NoSQL Database

Oracle NoSQL Database 是来自 Oracle Cloud Infrastructure (OCI) 的全托管云服务，提供按需吞吐量和高可用的存储配置。happyDomain 可将其作为可扩展的云端存储后端用于生产部署。

使用 Oracle NoSQL Database 需拥有 OCI 账户并创建 NoSQL 表。表需包含主键字段 `key`（字符串类型）和 `value` 字段（JSON 类型）存储数据。认证使用 OCI 的 IAM 和 API 签名密钥。

配置以下选项连接 happyDomain 至 Oracle NoSQL Database：

```
    -oci-compartment string
      	NoSQL 数据库所在的 OCI 隔间 ID
    -oci-fingerprint string
      	OCI 用户 API 密钥指纹
    -oci-private-key-file string
      	给定用户的 OCI 私钥文件路径
    -oci-region string
      	NoSQL 数据库所在的 OCI 区域 (默认 "us-phoenix-1")
    -oci-table string
      	存储值的表名 (默认 "happydomain")
    -oci-tenancy string
      	NoSQL 数据库所在的 OCI 租户 ID
    -oci-user string
      	访问 NoSQL 数据库的 OCI 用户 ID
```

#### 数据库管理系统

MySQL/Mariadb 等 DBMS 已不再支持，亦无相关计划。


持久化配置
-------------------

二进制文件会自动查找以下配置文件：

* 当前目录下的 `./happydomain.conf`；
* `$XDG_CONFIG_HOME/happydomain/happydomain.conf`；
* `/etc/happydomain.conf`。

仅使用找到的第一个文件。

也可通过命令行参数指定自定义路径：

```sh
./happyDomain /etc/happydomain/config
```

#### 配置文件格式

注释行必须以 # 开头，不支持行尾注释。

每行放置配置选项名称和期望值，用 `=` 分隔。例如：

```
storage-engine=leveldb
leveldb-path=/var/lib/happydomain/db/
```

#### 环境变量

还会查找以 `HAPPYDOMAIN_` 开头的特殊环境变量。

使用以下环境变量可达到与上述示例相同的效果：

```
HAPPYDOMAIN_STORAGE_ENGINE=leveldb
HAPPYDOMAIN_LEVELDB_PATH=/var/lib/happydomain/db/
```

只需将短横线替换为下划线即可。

#### 需要 OVH API？

OVH 没有简单的 API 密钥或凭据，需通过 Web 流程获取密钥。

启动认证流程，happyDomain 实例需配备专用应用程序密钥。

[连接 OVH，请按以下说明操作](https://help.happydomain.org/en/introduction/deploy/ovh)。


构建
--------

### 依赖项

构建 happyDomain 项目需具备以下依赖项：

* `go`；
* `nodejs`，已测试版本 22；
* `swag`，已测试版本 1.16（可通过 `go install github.com/swaggo/swag/cmd/swag@latest` 安装）。


### 构建步骤

1. 首先准备前端，安装 node 模块依赖：

```bash
pushd web; npm install; popd
```

2. 然后生成 Go 代码使用的资源文件：

```bash
go generate -tags swagger,web ./...
```

3. 最后编译 Go 代码：

```bash
go build -tags swagger,web ./cmd/happyDomain
```

此命令将创建独立二进制文件 `happyDomain`。


开发环境
-----------------------

若要为前端做贡献，而非每次修改后都重新生成前端资源（使用 `go generate`），可使用开发工具：

一个终端中使用以下参数运行 happydomain：

```bash
./happyDomain -dev http://127.0.0.1:5173
```

另一终端运行 node 部分：

```bash
cd web; npm run dev
```

此设置不使用集成到 go 二进制文件中的静态资源，而是将所有静态资源请求转发至 node 服务器，实现动态重载等功能。
