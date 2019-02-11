## Docker Stacks

This repository contains API definitions and implementations relating
to Docker Stacks, the runtime instantiation of Docker Compose based
applications.


### Standalone Runtime

The Standalone Stacks runtime is a full implementation of the Stacks API and
reconciler for Swarmkit stacks, intended to be ran as a separate container. It
communicates via the Swarmkit API via the local docker socket, and uses a fake
in-memory store for stack objects.

#### Building the standalone runtime

You may build the standalone runtime with

```
make standalone
```

#### Setting up the standalone runtime

The standalone runtime can be ran as a container on a swarmkit manager node:

```
docker run -v /var/run/docker.sock:/var/run/docker.sock -p 8080:2375 dockereng/stack-controller:latest
```
