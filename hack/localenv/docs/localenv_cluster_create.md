## localenv cluster create

Create a kinD cluster

### Synopsis

Create a kinD cluster and setup the greenhouse namespace optionally

```
localenv cluster create [flags]
```

### Examples

```
localenv cluster create --name <my-cluster> --namespace <my-namespace>
```

### Options

```
  -h, --help               help for create
  -c, --name string        create a kind cluster with a name - e.g. -c <my-cluster>
  -n, --namespace string   create a namespace in the cluster - e.g. -c <my-cluster> -n <my-namespace>
```

### SEE ALSO

* [localenv cluster](localenv_cluster.md)	 - Create, List and Delete kinD clusters

