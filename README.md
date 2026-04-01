# k2tf - Kubernetes YAML to Terraform HCL converter

[![Build Status](https://cloud.drone.io/api/badges/sl1pm4t/k2tf/status.svg)](https://cloud.drone.io/sl1pm4t/k2tf)
[![Go Report Card](https://goreportcard.com/badge/github.com/sl1pm4t/k2tf?)](https://goreportcard.com/report/github.com/sl1pm4t/k2tf)
[![Release](https://img.shields.io/github/release-pre/sl1pm4t/k2tf.svg)](https://github.com/sl1pm4t/k2tf/releases)

A tool for converting Kubernetes API Objects (in YAML format) into HashiCorp's Terraform configuration language.

The converted `.tf` files are suitable for use with the [Terraform Kubernetes Provider](https://www.terraform.io/docs/providers/kubernetes/index.html)

[![asciicast](https://asciinema.org/a/5LzAc7Eha7w7dwrktAxcMdpIc.svg)](https://asciinema.org/a/5LzAc7Eha7w7dwrktAxcMdpIc)

## Installation

**Pre-built Binaries**

Download Binary from GitHub [releases](https://github.com/sl1pm4t/k2tf/releases/latest) page.

**Build from source**

_See below_

**Homebrew**

```
$ brew install k2tf
```
**asdf**

Use the [asdf-k2tf](https://github.com/carlduevel/asdf-k2tf) plugin.


## Example Usage

**Convert a single YAML file and write generated Terraform config to Stdout**

```
$ k2tf -f test-fixtures/service.yaml
```

**Convert a single YAML file and write output to file**

```
$ k2tf -f test-fixtures/service.yaml -o service.tf
```

**Convert a directory of Kubernetes YAML files**

```
$ k2tf -f test-fixtures/
```

**Read & convert Kubernetes objects directly from a cluster**

```
$ kubectl get deployments -o yaml | ./k2tf -o deployments.tf
```

**Write each resource to its own file in a directory**

```
$ k2tf -f manifests.yaml -O output/
```

This creates one `.tf` file per resource in the output directory, named
`<resource_type>-<resource_name>.tf` (e.g. `kubernetes_deployment-my_app.tf`).

## CustomResourceDefinition (CRD) Support

Kubernetes `CustomResourceDefinition` resources are converted to
[`kubernetes_manifest`](https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs/resources/manifest)
Terraform resources. This approach preserves the full CRD manifest as a dynamic
object, which is necessary because CRD schemas are arbitrary and don't map to
static Terraform provider resource types.

```
$ k2tf -f crds.yaml
```

Example output:

```hcl
resource "kubernetes_manifest" "custom_resource_definition-crontabs_stable_example_com" {
  manifest = {
    "apiVersion" = "apiextensions.k8s.io/v1"
    "kind" = "CustomResourceDefinition"
    "metadata" = {
      "name" = "crontabs.stable.example.com"
    }
    "spec" = {
      ...
    }
  }
}
```

All other supported Kubernetes resource types continue to be converted to their
specific Terraform provider resources (e.g. `kubernetes_deployment_v1`,
`kubernetes_service_v1`, etc.).

## Building

> **NOTE** Requires a working Golang build environment.

This project uses Golang modules for dependency management, so it can be cloned outside of the `$GOPATH`.

**Clone the repository**

```
$ git clone https://github.com/sl1pm4t/k2tf.git
```

**Build**

```
$ cd k2tf
$ make build
```

**Run Tests**

```
$ make test
```

---

[![Downloads](https://img.shields.io/github/downloads/sl1pm4t/k2tf/total.svg)](https://img.shields.io/github/downloads/sl1pm4t/k2tf/total.svg)
