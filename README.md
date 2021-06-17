<p align="center">
  <img src="./docs/images/header.svg" width="838" />
</p>

## Introduction
Infra is **identity and access management** for Kubernetes. Provide any user fine-grained access to Kubernetes clusters via existing identity providers such as Okta, Google Accounts, Azure Active Directory and more.

**Features**:
* One-command access: `infra login`
* Fine-grained permissions
* Onboard & offboard users via Okta (Azure AD, Google, GitHub coming soon)
* Audit logs for who did what, when (coming soon)
* CLI & REST API
* Configure via `infra.yaml`

<p align="center">
  <img width="838" src="./docs/images/arch.svg" />
</p>

## Quickstart

### Install Infra Registry

```
kubectl apply -f https://raw.githubusercontent.com/infrahq/early-access/main/deploy/registry.yaml
```

Infra exposes a `LoadBalancer` service by default. Find the **External IP** of the load balancer:

```
kubectl get svc --namespace infra
```

### Install Infra CLI

```
curl -L "https://github.com/infrahq/early-access/releases/latest/download/infra-$(uname -s)-$(uname -m)" -o /usr/local/bin/infra && chmod +x /usr/local/bin/infra
```

### Log in

```
infra login <EXTERNAL-IP>
```

### Connect a Kubernetes cluster

First, retrieve your default Infra Registry API Key

```
infra apikey list
```

Then, install Infra Engine:

```bash
kubectl create namespace infra

kubectl create configmap infra-engine -n infra --from-literal="name=<CLUSTER NAME>" --from-literal="registry=<EXTERNAL IP>"

kubectl create secret generic infra-engine -n infra --from-literal="api-key=<API KEY>"

kubectl apply -f https://raw.githubusercontent.com/infrahq/early-access/main/deploy/engine.yaml
```

Verify the cluster has been connected:

```
infra destination list
```

To switch to this cluster, run

```
kubectl config use-context <CLUSTER NAME>
```

### Add users

* [Connect Okta](./docs/okta.md)
* [Add users manually](./docs/users.md)

### Map Permissions

To automatically assign permissions to specific users, create a config map containing the `infra.yaml` [configuration file](./docs/configuration.md).

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: infra
  namespace: infra
data:
  infra.yaml: |
    permissions:
      - user: michael@example.com
        destination: <CLUSTER NAME>
        role: edit
EOF
```

Then, restart Infra registry to apply the change:

```
kubectl rollout restart -n infra deployment/infra
```

## Documentation
* [Connect Okta](./docs/okta.md)
* [Add users manually](./docs/users.md)
* [Add a custom domain](./docs/domain.md)
* [CLI Reference](./docs/cli.md)
* [Configuration Reference](./docs/configuration.md)
* [Contributing](./docs/contributing.md)

## Security
We take security very seriously. If you have found a security vulnerability please disclose it privately to us by email via [security@infrahq.com](mailto:security@infrahq.com)
