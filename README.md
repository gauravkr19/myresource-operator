# myresource
myresource operator deploys a frontend app which performs CRUD operations and stores the data in Postgres database in Kubernetes, GKE.

## Description
### Operator features
Operator is developed using Operator SDK framework. It deploys the frontend CRUD app as a Deployment and a database as StatefulSet. The database is backed by a PVC. The PVC can be extended by passing `true` value to `pvcExtensionNeeded` field in CR or Helm values.yaml. The operator is packaged into Helm chart for easy deployment. 

### Initilaize operator-sdk
```
operator-sdk init --domain gauravkr19.dev --repo github.com/gauravkr19/myresource
operator-sdk create api --version v1alpha1 --kind MyResource --resource --controller
```
### CRUD App
The app has 4 APIs accessed internaly within the Pod's shell at the following path:
* `/api/create` for Create
* `/api/records` for Read
* `/api/update` for Update
* `/api/delete` for Delete

#### Request and Response
```
$ # CREATE
$ curl -X POST -H "Content-Type: application/json" -d '{"ID": 13, "AlertID": 45, "Label": "namespace", "Value": "crud-app"}' http://localhost:8080/api/create
Created record with ID 13 and AlertID 45

$ # READ
$ curl localhost:8080/api/records
{"id":13,"alertid":45,"label":"namespace","value":"crud-app"}

$ # UPDATE
$ curl -X POST -H "Content-Type: application/json" -d '{"ID": 13, "AlertID": 45, "Label": "namespace", "Value": "frontend"}' http://localhost:8080/api/update
Updated record with ID 13 and AlertID 45
$ curl localhost:8080/api/records
{"id":13,"alertid":45,"label":"namespace","value":"frontend"}

$ # DELETE
$ curl -X DELETE "http://localhost:8080/api/delete?num_records=1"
$ curl localhost:8080/api/records

```

### Helm values.yaml
Helm Chart requires values for  `dbUser` `dbPassword`. This is hardecoded in values.yaml but should be passed as `--set' field to not expose the secrets.
The Helm chart is served from Github Pages and the repo can be added using:
```sh
helm repo add crud-app https://gauravkr19.github.io/myresource-operator
helm search repo
```
The operator can be installed with command:

```sh
helm install crudapp crud-app/myresource-operator  -f docs/charts/values.yaml  --namespace crudapp --create-namespace
```

The operator and the apps are installed in `crudapp` namespace created by helm, else will be deployed in `default` namespace
The installation fails without the values of `bUser`, `dbPassword` passed via `values.yaml`. These values are hardcoded for demo purpose.


### Important paths listed for quick reference
* Helm Chart files `https://github.com/gauravkr19/myresource-operator/tree/main/docs/charts`.
* controller.go `https://github.com/gauravkr19/myresource-operator/blob/main/controllers/myresource_controller.go`.
* types.go `https://github.com/gauravkr19/myresource-operator/blob/main/api/v1alpha1/myresource_types.go`.
* crudapp `https://github.com/gauravkr19/crudapp`



## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Running on the cluster
1. Install Instances of Custom Resources:

```sh
kubectl apply -f config/samples/
```

2. Build and push your image to the location specified by `IMG`:

```sh
make docker-build docker-push IMG=<some-registry>/myresource:tag
```

3. Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=<some-registry>/myresource:tag
```

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller
UnDeploy the controller from the cluster:

```sh
make undeploy
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/),
which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.

### Test It Out
1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
