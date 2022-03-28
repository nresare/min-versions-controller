# min-versions controller

http://github.com/nresare/min-versions

This project implements a mechanism where labels on a Pod can be used to enforce
certain minimum versions of the kubelet and containerd software on the Nodes that
are eligible to schedule the Pod.

Once the mechanism is deployed onto a cluster, the labels
`mwam.com/min-containerd-version=1.2`, `mwam.com/min-kubelet-version=1.4` added
to a Pod would prevent versions of containerd and kubelet lower than 1.2 and 1.4
respectively.

It is implemented using two mechanisms separate mechanisms working together:
* A controller that observes all Node objects in the cluster and maintains 
  labels on the Node such that the affinity mechanism can be used to exclude
  Nodes with too old a version of either of the pieces of software.
* A mutating webhook intended to modify any Pod object before it gets created
  in the cluster such that if the above-mentioned labels are present, their
  semantics gets converted into the appropriate affinity field in the Pod spec
  that will prevent scheduling on nodes with non-compliant software.

Both of these two components are implemented using the same Deployment.

## Deploying

The original intention was to use the provided [kind-bootstrap](https://github.com/MarshallWace/kind-bootstrap)
project, but since it turned out that that project did not support running
natively on my Apple Silicon (arm64) computer, I ended up forking the project
and setting up a multi architecture version of the setup [here](https://github.com/nresare/kind-bootstrap/tree/arm64_support)

The steps to deploy are a bit manual, mostly since it seems like a bad habit
to ship private certificate keys. They have been tested on a fully up-to-date
macOS 12.3 fully up-to-date Ubuntu 20.04 on an amd64 EC2 instance.

Prequisites:
* docker (on ubuntu, I used the 20.04 `docker.io` package, on macOS [Docker desktop](https://www.docker.com/products/docker-desktop/))
* kubectl (on ubuntu, [instructions here](https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/) on macOS `brew install kubectl`) 
* kind (on ubuntu, [instructions here](https://kind.sigs.k8s.io/docs/user/quick-start/#installing-from-release-binaries) on macOS `brew install kind`)

Steps to get the system up and running :
```sh
$ git clone https://github.com/nresare/kind-bootstrap
$ cd kind-bootstrap && git checkout arm64_support
$ ./kind-with-local-registry.sh
$ kubectl create namespace min-versions
$ cd ..
$ git clone https://github.com/nresare/min-versions-controller
$ cd min-versions-controller
$ docker build . --tag localhost:5000/min-versions-controller:v0.1.1
$ docker push localhost:5000/min-versions-controller:v0.1.1
$ cd cert && ./gencert.sh
$ base64 -w0 ca.crt && echo  # on macOS, skip the -w0 option
$ # copy the base64 encoded ca-cert to the clipboard buffer
$ cd ..
$ $EDITOR controller.yaml
$ # replace the value of webhooks[0]:clientConfig:caBundle with the cert from the clipboard buffer
$ kubectl apply -f controller.yaml
$ kubectl logs -n min-versions -l app.kubernetes.io/name=min-versions-controller # to verify that the workload runs 
```

Things to do to verify that everything works as expected:
* Verify with `kubectl get nodes --show-labels` that the Node objects are correctly labelled
* `kubectl apply -f test-pod.yaml` will bring up a Pod that can only be scheduled on `mw-worker`
* Once the `default/dbg` pod is deployed, inspect the `spec:affinity:nodeAffinity` block with `kubectl get pod dbg -o yaml`
* A quick way of seeing where a pod is deployed `kubectl get pods -o wide`

## Known issues

* When attempting to update either of the labels on a Pod that controls the Node
  selection, the mutating hook will fail to modify the affinity field of the Pod
  spec and the update will fail as the affinity field is read only after creation.
* The whole setup would be helped by being driven from something like a build 
  system.

## How to build the container

```sh
$ docker build . --tag localhost:5000/min-versions-controller:$VERSION
$ docker push localhost:5000/min-versions-controller:$VERSION
```

To build the container for multiple platforms deployed to a remote
container registry, use the following commands. [buildx](https://github.com/docker/buildx)
needs to be configured according to the instructions for this to work.

```sh
$ docker buildx build --platform linux/amd64,linux/arm64 \
  -t $REPO/min-versions-controller:$VERSION . --push
```

## Debugging TLS

To get a shell with curl in the current cluster, use
`kubectl run -it dbg --image=nresare/dbg --rm -- sh`

Once inside, you can verify the TLS setup with
```sh
$ cat > ca.crt
<paste from certs/ca.crt>
ctrl-d
$ curl -v --cacert ./ca.crt https://podhook.min-versions.svc
```

## Tailing the controller logs

```sh
$ logs -f -l app.kubernetes.io/name=min-versions-controller -n min-versions
```