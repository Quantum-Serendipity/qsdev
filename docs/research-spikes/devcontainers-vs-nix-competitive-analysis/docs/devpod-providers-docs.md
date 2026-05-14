# DevPod — Providers Documentation (Official)
- **Source**: https://devpod.sh/docs/managing-providers/what-are-providers and https://devpod.sh/docs/managing-providers/add-provider
- **Raw GitHub sources**: loft-sh/devpod/docs/pages/managing-providers/
- **Retrieved**: 2026-03-20

## What are Providers?

Providers are simple CLI programs that let DevPod create, manage and run the workspaces requested by the user. In the simplest form, a provider defines a command to create, delete and connect to a virtual machine in a cloud.

DevPod relies on the provider model in order to allow flexibility and adaptability for any backend of choice. Providers in DevPod are defined through a `provider.yaml` that defines the necessary options, configuration, binaries and commands needed to handle workspace creation.

## Type of Providers

### Machine Providers

Machine providers are those who will create and manage a VM for the workspace selected. An example of a Machine provider is the AWS Provider, which uses EC2 instances to run the environment. These type of providers will also manage the lifecycle of the VM, starting/stopping and deleting it when needed.

### Non-Machine Providers

Non-machine providers are those who will work directly with containers, instead of using VMs. An example of this can be the SSH, Kubernetes and Docker providers. Those providers will **not** create any VMs, but instead directly run the workspace container on the target.

## Official Providers (maintained by DevPod team)

- [Docker](https://github.com/loft-sh/devpod/tree/main/providers/docker) — local Docker daemon
- [Kubernetes](https://github.com/loft-sh/devpod-provider-kubernetes) — any k8s cluster
- [SSH](https://github.com/loft-sh/devpod-provider-ssh) — any reachable remote machine
- [AWS](https://github.com/loft-sh/devpod-provider-aws) — EC2 instances
- [Google Cloud](https://github.com/loft-sh/devpod-provider-gcloud) — GCE instances
- [Azure](https://github.com/loft-sh/devpod-provider-azure) — Azure VMs
- [Digital Ocean](https://github.com/loft-sh/devpod-provider-digitalocean) — Droplets

## Community Providers

- Cloudbit (cloudbit-ch/devpod-provider-cloudbit)
- Flow (flowswiss/devpod-provider-flow)
- Hetzner (mrsimonemms/devpod-provider-hetzner)
- OVHcloud (alexandrevilain/devpod-provider-ovhcloud)
- Scaleway (dirien/devpod-provider-scaleway)
- Exoscale (dirien/devpod-provider-exoscale)
- Multipass (minhio/devpod-provider-multipass)
- Open Telekom Cloud (akyriako/devpod-provider-opentelekomcloud)
- Vultr (navaneeth-dev/devpod-provider-vultr)
- STACKIT (stackitcloud/devpod-provider-stackit)

## Provider Installation

```sh
devpod provider add docker
devpod provider add kubernetes
devpod provider add ssh
devpod provider add aws
# etc.
```

Multiple providers of the same type with different options:
```sh
devpod provider add aws --name aws-gpu -o AWS_INSTANCE_TYPE=p3.8xlarge
```

Custom providers from GitHub:
```sh
devpod provider add loft-sh/devpod-provider-terraform
devpod provider add my-org/my-repo@v0.0.1
```

From local path or URL:
```sh
devpod provider add ../my-provider/provider.yaml
devpod provider add https://example.com/provider.yaml
```

## Single Machine Provider

By default, DevPod will use a separate machine for each workspace. You can enable "Reuse machine" to use a single machine for all workspaces:
```sh
devpod provider use <provider-name> --single-machine
```

## Provider Options

Each provider has configurable options. Example for AWS:
- AWS_ACCESS_KEY_ID
- AWS_SECRET_ACCESS_KEY
- AWS_REGION (required)
- AWS_INSTANCE_TYPE (default: c5.xlarge)
- AWS_DISK_SIZE (default: 40 GB)
- AWS_AMI
- AWS_VPC_ID
- INACTIVITY_TIMEOUT (default: 10m)
- INJECT_DOCKER_CREDENTIALS (default: true)
- INJECT_GIT_CREDENTIALS (default: true)
