# Shootout

Project is designed to be run in local k8s cluster.

Developed and tested on M1 Mac 12.3.1 with default Docker Desktop k8s cluster.

## Run
```
# build docker images for local k8s cluster
make build

# install CRD to your local k8s cluster and launch controller
cd operator
make build
make install
make run

# in other terminal add sample resource, from project root do
kubectl apply -f operator/config/samples/cowboys_v1_shootout.yaml

# observe logs
kubectl logs --max-log-requests 7 --prefix --pod-running-timeout=20s -f -l shootout=shootout-sample --all-containers
```

## Clean Up
```
kubectl delete shootouts.cowboys.mejeras.lt shootout-sample
cd operator
rm bin/kustomize
make uninstall
```

## Tests
```
make test
```
