## Cluster-Cluster Trust Setup in KinD Cluster

Kubernetes issued Service Account tokens are JWT compliant and can be used as OIDC tokens.

This essentially means that we can have a cluster introspect the token of another cluster and trust it.

### Caveats in KinD Cluster

- KinD cluster issued tokens have https://kubernetes.default.svc as the issuer and this cannot be introspected by other
  clusters as it would reach its own APIServer
- Furthermore, the JWKS URI endpoint `/openid/v1/jwks` has an IP address of the docker network and cannot be reached by
  other cluster's APIServer

We can overcome this limitation by modifying the

- service account issuer
- service account JWKS URI endpoint

### Step 1: Modify the Service Account Issuer and JWKS URI of greenhouse admin cluster

We can modify the issuer and JWKS URI of the greenhouse-admin-cluster by using KinD configuration

Create the admin cluster with the following configuration:

```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
    kubeadmConfigPatches:
      - |
        kind: ClusterConfiguration
        apiServer:
          extraArgs:
            "anonymous-auth": "true"
            "service-account-issuer": "https://greenhouse-admin-control-plane:6443"
            "service-account-jwks-uri": "https://greenhouse-admin-control-plane:6443/openid/v1/jwks"
```            

> [!NOTE]
> `https://greenhouse-admin-control-plane:6443` is the internal docker network address of the greenhouse admin cluster (
> Provided you create the cluster with the name greenhouse-admin)

```shell
kind create cluster --name greenhouse-admin --config greenhouse-admin-config.yaml
```

### Step 2: Retrieve the CA from kube-root-ca.crt ConfigMap in the greenhouse admin cluster

Since the APIServer CA is self-signed, we need to provide this CA to the remote cluster. You can find this config in the
`default` namespace.

> [!NOTE]
> Kubernetes will create this ConfigMap in every namespace.

```shell
# Retrieve the CA from the kube-root-ca.crt ConfigMap and save it to a file in TMP folder
kubectl get cm kube-root-ca.crt -n default -o json | jq -r '.data."ca.crt"' > $TMPDIR/greenhouse-admin-ca.crt
```

We will need this later while creating the remote cluster.

### Step 3: Expose the Discovery Endpoint of the greenhouse admin cluster

In order for a successful introspection, the remote cluster needs to reach the JWKS URI of the greenhouse admin cluster.

The JWKS URI is resolved from `/.well-known/openid-configuration` endpoint and it must be unauthenticated endpoint.

```shell
kubectl get --raw /.well-known/openid-configuration | jq

{
  "issuer": "https://greenhouse-admin-control-plane:6443",
  "jwks_uri": "https://greenhouse-admin-control-plane:6443/openid/v1/jwks",
  "response_types_supported": [
    "id_token"
  ],
  "subject_types_supported": [
    "public"
  ],
  "id_token_signing_alg_values_supported": [
    "RS256"
  ]
}

```

`KinD` cluster has a ClusterRole called `system:service-account-issuer-discovery` and you can create a
ClusterRoleBinding with the Group `system:unauthenticated` to allow unauthenticated access to the discovery endpoint.

```shell
kubectl create clusterrolebinding oidc-reviewer-binding --clusterrole=system:service-account-issuer-discovery --group=system:unauthenticated
```

### Step 4: Create the remote cluster

Create the remote cluster with the following configuration:

```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
    kubeadmConfigPatches:
      - |
        kind: ClusterConfiguration
        apiServer:
          extraArgs:
            "oidc-issuer-url": "https://greenhouse-admin-control-plane:6443"
            "oidc-client-id": "greenhouse"
            "oidc-username-claim": "sub"
            "oidc-groups-claim": "groups"
            "oidc-username-prefix": "greenhouse:"
            "oidc-ca-file": "/etc/kubernetes/pki/oidc-ca.crt"  # Trust the admin cluster CA
            "anonymous-auth": "true"
    extraMounts:
      - hostPath: <absolute/path/to/greenhouse-admin-ca.crt>
        containerPath: /etc/kubernetes/pki/oidc-ca.crt
        readOnly: true
```

> [!NOTE]
> echo $TMPDIR will give you the path to the tmp directory where we saved the `greenhouse-admin-ca.crt` file

```shell
kind create cluster --name greenhouse-remote --config greenhouse-remote-config.yaml
```

> [!TIP]
> Once the remote cluster is created, you can verify the OIDC setup by looking for errors in the `apiserver` pod logs
> `kubectl logs -n kube-system -l component=kube-apiserver` if there are errors then you will find logs starting with
`OIDC`

### Step 5: Verify the OIDC setup

Create a ClusterRoleBinding `greenhouse-admin-access` in the remote cluster and bind the `cluster-admin` ClusterRole to
the user `greenhouse:system:serviceaccount:oidc:default`

```shell
kubectl create clusterrolebinding greenhouse-admin-access --clusterrole=cluster-admin --user=greenhouse:system:serviceaccount:oidc:default
```

Create a namespace in the admin cluster called `oidc`

```shell
kubectl create namespace oidc
```

In the Admin cluster context, create a token request for the service account `default` in the `oidc` namespace with the
audience `greenhouse`

```shell
export TOKEN=$(kubectl create token default -n oidc --audience greenhouse)
```

Now try to access the remote cluster

```shell
kubectl get pods -A --token=$TOKEN --server=<remote-cluster-APIServer-URL> --insecure-skip-tls-verify=false

NAMESPACE            NAME                                                      READY   STATUS    RESTARTS   AGE
kube-system          coredns-668d6bf9bc-5htgn                                  1/1     Running   0          19m
kube-system          coredns-668d6bf9bc-vdfn9                                  1/1     Running   0          19m
kube-system          etcd-greenhouse-remote-control-plane                      1/1     Running   0          19m
kube-system          kindnet-mhwbv                                             1/1     Running   0          19m
kube-system          kube-apiserver-greenhouse-remote-control-plane            1/1     Running   0          19m
kube-system          kube-controller-manager-greenhouse-remote-control-plane   1/1     Running   0          19m
kube-system          kube-proxy-vlxwj                                          1/1     Running   0          19m
kube-system          kube-scheduler-greenhouse-remote-control-plane            1/1     Running   0          19m
local-path-storage   local-path-provisioner-58cc7856b6-rlgtq                   1/1     Running   0          19m
```

> [!NOTE]
> We are using the `--insecure-skip-tls-verify=false` just for demo purposes.
> In production, you should not use the `--insecure-skip-tls-verify` flag, instead you should pass the flag
> --certificate-authority=<path-to-remote-cluster-ca.crt>
> the CA file can be found in ConfigMap `kube-root-ca.crt` in the `default` namespace of the remote cluster



