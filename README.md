WARNING: This is in the early stages of development, do not use it yet - you will not have a good experience.

DistributedCD (DCD) is a platform for distributed pipelines (aka CI/CD). It is designed to produce repeatable builds with a centralised and durable build history, but where the execution itself is run locally to give greater immediacy than a typical CI/CD setup.

This repo contains the runner, which is packaged and distributed as a docker image. The runner is a single standalone binary at `/dcd`, which is all the image contains. It is intended to be used as a base image, extended with the dependencies of your CI/CD pipeline. The runner handles interpreting a JSON pipeline definition, executing the steps in the pipeline and persisting the results.

# Usage

## Create reusable image for running pipeline

Create a docker image that is available to each member of the team. When building it is recommended to tag with `latest` so that team members can use this to always get the latest version when running the pipeline.

```Dockerfile
FROM ghcr.io/progsoftware/dcd:v0.1.0 AS dcd

FROM your-base-image

COPY --from dcd /dcd /dcd

# Set up image for CI/CD

ENTRYPOINT ["/dcd"]
```

The suggest naming convention for your pipeline image is`ghcr.io/your-org/your-project-dcd:latest`. We will store the exact sha256 image digest tag, so it is suggested not to bother trying to create another unique version for each build - just use `latest`.

```
docker build -t ghcr.io/your-org/your-project-dcd:latest
```

## Running a pipeline

Note that access to the host docker socket is required to get information about the container and image that ran the pipeline (for recording purposes). Docker is used to get a repeatable software stack, not to try to contain untrusted code. It is expected that pipelines will being using Docker and that the code will be highly trusted.

The following is the standard set of options to docker and should show a usage message - replace `ghcr.io/your-org/your-project-dcd:latest` with the image name built and pushed based on the above:

```shell
docker run -v /var/run/docker.sock:/var/run/docker.sock -v $PWD:$PWD -w $PWD --rm -it ghcr.io/your-org/your-project-dcd:latest
```
 
For further invocations commands and options can be added after the image name.

To simplify this for a given project, set up a script to run it:

```shell
# from project root
cat <<'END' > dcd
#!/bin/sh
set -e
docker run -v /var/run/docker.sock:/var/run/docker.sock -v $PWD:$PWD -w $PWD --rm -it ghcr.io/your-org/your-project-dcd:latest "$@"
END

chmod +x dcd

./dcd # <- should output usage message
```



