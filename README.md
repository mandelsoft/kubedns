# kubedns

An environment to manage hosted zones and appropriate authoritative name servers for the Domain Name System with Kubernetes resources.

## Description

This project provides a controller-manager managing the deployment of
[kubedyndns](https://github.com/mandelsoft/kubedyndns) DNS servers configured as slave servers to manage DNS entries for a hosted zone. Hosted zones, as well as DNS records in those zones are described by Kubernetes resources `HostedZone` and `CoreDNSEntry`  in namespaces. The DNS server is a [coredns](https://github.com/coredns/coredns) server known from Kubernetes enriched with the *kubedyndns* plugin able to serve DNS records defined by Kubernetes resources.

For every root zone such a dns server is deployed acting as *Authoritative Name Server* for this zone and locally configured nested zones.

## Basic Architecture

The controllers are implementated with the [kubecrtutils](https://github.com/mandelsoft/kubecrtutils) library, which is a wrapper arround the multi-cluster runtime library supporting logigal clusters and fleets.

### User-facing API

This system uses two custom resources `HostedZone`and `CoreDNSEntry` to configure authoritative name servers.
`HostedZone` objects can be maintained in namespaces and describe a hosted zone with all its properties.
`CoreDNSEntriy` objects relate to a hosted zone in the same namespace and describe domains known to this zone.
Supported are A, CNAME, TXT, SRV and NS records.
An entry object may describe multiple record
types. A namespace may define multiple related or unrelated hosted zones.

Nested zones can be configured by NS records referring to another name server, or by a nested local hosted zone. Those zone objects have a parent reference to their parent zone object.

Every zone or entry object may define multiple domains. Hereby, only the root zone defines  FQDNs, nested zones and records define names relative to their zone.

This way a zone object can formally describe multiple identical zones with different root domains.
A record then finally describes a set of domain names according to the cartesian product ( R x N ... N x E) of the name sets along the nesting hierarchy (R Root zone, N sequenece of nested zones, E entry object).
The final set of names will be reported in the status of the entry objects.

### Implementation 

For every root zone object a separate DNS server is deployed, it serves all the configured base domains and all the locally nested zones, also.

The deployment consists of a Kubernetes `Deployment` and a `Service` used to expose the DNS port via a load-balancer (for UPD no shared ingress load-balancer can be used)
For the deployment an own `ServiceAccount` and RBAC rules are created to restrict the access of the server to the namespace of the hosted zone object.

If a separate runtime cluster is used additionally a service account `Secret` is created in the API namespace, which is propagated to the runtime cluster to enable the access to the namespace in the dataplane.

## Operational modes

The controller manager provides three pairs of controllers used to implement
different scenarios:
- *direct scenario*
  The controller group `operator` provides controllers directly implementing the
  DNS provider scenario.

  They act on two logical clusters:
  - `dataplane`: the objectspace holding the user facing resources. It might be backed by a regular Kubernetes cluster or a fleet (like a KCP fleet).
  - `runtime`: the Kubernetes cluster used to deploy the DNS servers. This must be backed by a Kubernetes cluster.
  
- *replication scenario*
  The controller group `replication` provides controllers usable to replicate user facing resources into another dataplane. 
  
  They act on two logical clusters:
  - `source`: the objectspace holding the user facing resources. It might be backed by a regular Kubernetes cluster or a fleet (like a KCP fleet).
  - `target`: The target Kubernetes cluster to replicate to. This must be backed by a Kubernetes cluster.

  This groupo map be combined with group `operator`, to work locally on the target cluster, or with the group `server`.

- *REST server scenario*
  The controller group `server` provides controllers used to feed a REST server to offer query operations for FQDNs in the context of a hosted zone defined
  by a cluster identity, namespace and object name of the hostedzone object.

  They act on one logical cluster:
  - `source`: the objectspace holding the user facing resources. It might be backed by a regular Kubernetes cluster or a fleet (like a KCP fleet).

The controller manager offers the following logical clusters:
- `source`: cluster for the user facing API. It falls back to cluster `target`. It might be backed by a fleet.
- `target`: a replication target. It uses the default cluster as fallback.
- `runtime`: the runtime cluster used to deploy DNS servers. It falls back to `dataplane`.
- `dataplane`: the API cluster used to control the DNS server deployment. It falls back to cluster `target`.

Those clusters are directly mapped to the controller clusters.

### API/Runtime Organization

The operator is able to work with different operational modes
requiring one or two Kubernetes dataplanes.
- *Vanilla Mode*: A single cluster is used for the end-user (this is the API) to configure zones and records and to deploy the runtime for the dns servers. The dns servers and required additional resources are deployed into the user namespace. API users should only have permissions for the dns resources.

- *Distinguished Vanilla Mode*: The runtime resources are deployed into a separate runtime namespace not accessible by the end users

- *Multi-Cluster Mode*: The Kubernetes dataplane used by the end-users to configure their zones and records is separated from the runtime cluster used to deploy the server runtime.
  This API dataplane can be a nodeless Kubernetes cluster just used as API for manageing zones and records (for example a [Gardener](https://gardener.cloud) Nodeless Shoot), but also any other user-facing cluster.
   The deployment of such a combination of API-server, etcd and kube-controller-manager into the runtime cluster is possible but not supported by this project, yet.
    
    The runtime cluster can be configured to use a single namespace to deploy the implementation resources, or
    to use dedicated namespace shadowing the API namespaces.

### DNS support accessing the Nameservers

A nameserver requires a CNAME to be configurable for an NS record to bind it into a DNS tree.

The controller provides an interface for provisioning such names. There are three implementations part of this code base.

- `loadbalancer`: If the UDP load-balancer configured for a service provides CNAMEs, such names can directly be used.
  But it is not guaranteed that this name is stable after recreating the service (in case of some lost resources)
- `gardener`: Gardener supports an out-of-the-box DNS sub-domain exclusively for a cluster. This mode configures a stable CNAME unique for every root hosted zone under this domain.
- `kubedns`: It uses its own resources to publish a CNAME.
  Therefore, a separate explicit deployment of a `kubedyndns` service is required (similar to the ones managed by this controller-manager). This service must be bound to some accessible CNAME and added via NS records to some DNS tree used in the local environment.
    It uses a reserved API namespace `dns-system` to maintain the DNS records for the managed name servers and must use a different `class`.
    for its hosted zones to define a separate management responsibility for such hosted zones.

### Sharding

The controller is prepared for controller-sharding in two dimensions.

- a `class` attribute can be used to run multiple service environments in one API dataplane. Every environment should then use its own runtime environments and controllers.
- a `runtime` attribute can be used together with a scheduler (not included) to support the distribution of the dns servers into multiple runtime clusters. In this case controllers must be started separately for every runtime cluster with different non-empty runtime attributes set.
  This feature can be used to manage a highly scalable environment. The automatic management of such an environment is not included in this project. 
- 
## IaaS Support

Special support is currently available for AWS, requiring
a dedicated load-balancer (NLB) type to serve UDP requests.

If other environments also require such a special handling,
there is an interface to plug-in such support, but it is not available as part of this project (Feel free to contribute).

## REST-Server API

The REST servers offers an API (API version v1) for looking up 

- full qualified domain names (`api/v1/zones/<cluster id>/<namespace>/<zone name>/<fqdn>`)

  This request also handles local nested zones (definaed by nested hosted zone objects)
  It returns an `error` answer for IP addresses not in the domain, or an `info` answer, which my provide an empty record list, if there is no such record. The `names`field provide the appropriate matching FQDN. 
  
  
- a reverse lookup for IP addresses (`api/v1/ips/<cluster id>/<namespace>/<zone name>/<ipv1 or ipv6>`)

  It returns an `error` answer for IP addresses not in the domain, or a list of `info` documents, for every matching entry. the `names` field may contain multiple entries, if multiple names are declared in the resource objects. THis gives a complete list of matching names for the IP address.


### Answer formats

The answer is always a single JSON document.

#### `error`

In case of an error a JSON document with a field `error` is returned, which contain the error message.

#### `info`

An `info` document includes the following fields:

- `zone`: a zone attribute map with the fields:
  - `names` *[]string*: the list of matching zone domains (Pleate not, that a hosted zone object might declare more than one DNS name).
  - `email` *string*: the e-mail address
  - `minimumTTL` *int*:  the minimul TTL property of the zone
  - `expire` *int*: the expiration property of the zone
  - `refresh`*int*: the refresh property of the zone
- `names` *[]string*: the list of matching FQDNs.
- `records` *[]record*: the list of found records

#### `record`

A `record` document describes a list of records matching of the same type and includes the following fields:

- `A` *[]ipv4*: list of IPv4 addresses
- `AAAA` *[]ipv6*: list of IPv6 addresses
- `TXT` *[]string*: list of text records
- `SRV` *[]service*: list of services
- `ǸS` *[]string*: list of name servers
- `CNAME` *string*: CName entry

#### `service`

A `service` document describes a service entry with the following fields:

- `service` *string*: the service name
- `records`*[]srvrecord*: list of service record entries

#### `srvrecord`

This is the service description:

- `protocol` *string*: UPD or TCP
- `priority` *int*: the priority property
- `weight` *int*: the weight property
- `port` *int*: the port number
- `host` *string*: the target name (FQDN)
- 


## Options

```
Usage of dnsmanager:
      --class *string                               name of the controller class to handle (default <none>)
      --controllers strings                         activated controllers (corednsentry, hostedzone, replication.corednsentry, replication.hostedzone, server.corednsentry, server.hostedzone, all, operator, replication, server). (default [all])
      --dataplane-kubeconfig string                 api cluster for functional controllers
      --dataplane-kubeconfig-context string         context used together with dataplane-kubeconfig
      --dataplane-kubeconfig-endpointslice string   endpointslice used together with dataplane-kubeconfigfor APIExport
      --dataplane-kubeconfig-identity string        identity used together with dataplane-kubeconfig
      --dns-class string                            DNS class for managed nameserver DNS names (default "dns-system")
      --dns-domain string                           DNS domain for managed nameserver DNS names
      --dns-mode string                             DNS mode for providing nameserver cnames [gardener,kubedns,loadbalancer] (default "loadbalancer")
      --dnsapi string                               The port on which to run the DNS API server. (default ":8085")
      --enable-http2                                If set, HTTP/2 will be enabled for the metrics and webhook servers
      --health-probe-bind-address string            The address the probe endpoint binds to. (default ":8081")
      --iaas string                                 IaaS layer to use (special support so far for "aws" (default "default")
      --kubeconfig string                           path to standard kubeconfig
      --kubeconfig-context string                   context used together with kubeconfig
      --kubeconfig-identity string                  identity used together with kubeconfig
      --leader-elect                                Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.
      --leader-elect-namespace string               leader election namespace
      --leader-election-id string                   Id for leader election
      --log-level string                            logging level (default "info")
      --log-rule stringToString                     logging rules (default [])
      --metrics-bind-address string                 The address the metrics endpoint binds to. Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service. (default "0")
      --metrics-cert-key string                     The name of the metrics server key file. (default "tls.key")
      --metrics-cert-name string                    The name of the metrics server certificate file. (default "tls.crt")
      --metrics-cert-path string                    The directory that contains the metrics server certificate.
      --metrics-secure                              If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead. (default true)
      --ns-namespace string                         namespace used to request nameserver DNS names (default "dns-system")
      --runtime *string                             name of the runtime to handle (default <none>)
      --runtime-kubeconfig string                   runtime cluster
      --runtime-kubeconfig-context string           context used together with runtime-kubeconfig
      --runtime-kubeconfig-identity string          identity used together with runtime-kubeconfig
      --runtime-namespace string                    use single runtime namespace for deployments
      --server-class *string                        class name to be served by REST server (default <none>)
      --slave-mode                                  run server in slave mode
      --source-kubeconfig string                    user api cluster
      --source-kubeconfig-context string            context used together with source-kubeconfig
      --source-kubeconfig-endpointslice string      endpointslice used together with source-kubeconfigfor APIExport
      --source-kubeconfig-identity string           identity used together with source-kubeconfig
      --target-class string                         target class for replication
      --target-kubeconfig string                    replication target
      --target-kubeconfig-context string            context used together with target-kubeconfig
      --target-kubeconfig-identity string           identity used together with target-kubeconfig
      --target-namespace string                     namespace used to request nameserver DNS names
      --workers stringToInt                         workers for controller (default [])
```

## System Setup

### Single Cluster Setup

Deploy CRDs, the dataplane and runtime RBAC objects into the cluster. Afterwards, deploy the dataplane controller-manager into the `dns-system` namespace
using in-cluster access for the default cluster (no kubeconfig options)

If you add the `runtime-namespace` option to the controller, the service deployments with be centrally done in this namespace (typically use `dns-runtime`)

### Multi-CLuster Setup

Deploy CRDS and the dataplane RBAC objects to your dataplane cluster
and the runtime RBAC objects into the runtime cluster(s)

Then you must create a kubeconfig for the service account token created in the `dns-system` namespace and add it to a secret (`controller-manager-dataplane`) in the `dns-system` namespace of the runtime cluster (`KUBECONFIG=<dataplane kubeconfig> bin/create-kc -r <runtime kubeconfig>`)

Afterwards, deploy the runtime controller-manager into the `dns-system` namespace of your runtime cluster

### Local Setup

You can run the controller locally using your system kubeconfigs.
To run the controller with the same access permissions than a deployed one,
you can create the appropriate kubeconfigs with `bin/create -c <clustername> -a` with the appropriate kubeconfigs (if your kubeconfig files are named `dns.runtime` and/or `dns.apii` you can omit option `-c`) 

 ### Prerequisites
- go version v1.24.6+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to one or two Kubernetes v1.11.3+ cluster.


## License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

