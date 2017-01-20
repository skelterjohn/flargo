# flargo

Resumable dynamic workflows on top of Google Container Builder (or cloudbuild for short).

## how it works

The cloudbuild service provides units of execution as a series of steps. This unit of execution is called a "build" by the cloudbuild service, but in reality it's an arbitrary execution with some tooling to make it easy to use for building container images.

For a build to succeed, every step in the build must succeed. If a step fails, you cannot skip it or retry.

`flargo` creates a workflow pipeline on top of the cloudbuild service.

In a `flargo` config file, you specify a set of builds and their dependencies. The `flargo` command line will use the cloudbuild service to run these builds, where the first step is "wait for my dependencies". At any point a particular build can be retried or skipped.

When builds in the workflow complete, they publish on Google Cloud Pub/Sub (or pubsub for short). Other builds wait for their dependencies by subscribing to the workflow pubsub topic.

Skipping is done by canceling a build (if needed) and sending the `done` message on pubsub directly. Retrying a build is done by canceling the previous attempt (if needed) and creating a new build that will send the message when complete.

You can keep track of a particular `flargo` workflow by using the workflow ID. This ID corresponds to a cloudbuild build ID that is used as a kickoff point for execution, and that build's logs provide information to the `flargo` tool in order to allow it to manage things later.

Any docker images built by a `flargo` build will be pulled into the next builds in the pipeline.

If a `flargo` build, files to be sent to the next builds need to be written to a directory named `out`. Files from earlier builds will be available in `in/$EXECUTION_NAME`. `flargo` will store these intermediate files in Google Cloud Storage(GCS).

The current directory will be sent as the source for each `flargo` build, with the `in` and `out` directories put in afterwards (so don't use those directories).

## auth and project settings

`flargo` bootstraps on `gcloud` auth and its project property.

## terminology

Each `flargo` "item" is called an "execution". An execution corresponds to a
single cloudbuild build.

A `flargo` config is a list of executions and their dependencies.

## config

### working example
```
# build will begin immediately.
exec: build() {
steps:
- name: 'gcr.io/cloud-builders/golang-project:alpine'
  args: ['service', '--tag=gcr.io/$PROJECT_ID/service']
images: ['gcr.io/$PROJECT_ID/service']
}

# build will begin immediately.
exec: build_probes() {
steps:
- name: 'gcr.io/cloud-builders/docker'
  args: ['build', '--tag=gcr.io/$PROJECT_ID/probes', 'probes']
images: ['gcr.io/$PROJECT_ID/probes']
}

# "deploy_to_dev" will only start once "build" is complete. Its "out" files will
# be put in "in/build".
# This example assumpes that retagging an image is sufficient to deploy. That
# is, it assumes that the runtimes are watching for pushes to specific tags.
exec: deploy_to_dev(build) {
steps:
- name: 'gcr.io/cloud-builders/docker'
  args: ['tag', '-f', 'gcr.io/$PROJECT_ID/service', 'gcr.io/$PROJECT_ID/service:dev']
images: ['gcr.io/$PROJECT_ID/service:dev']
}

# Once dev is ready, run the probes (built earlier) to check that things are
# working.
exec: test_dev(deploy_to_dev, build_probes) {
steps:
- name: 'gcr.io/$PROJECT_ID/probes'
  args: ['dev']
}

# A wait directive means that a human has to run flargo to indicate this
# execution has completed. It's used as a manual gate between dev and prod,
# in this example.
wait: dev_to_prod(test_dev) {}

# "deploy_to_prod" will only start once "dev_to_prod" is complete.
exec: deploy_to_prod(dev_to_prod) {
steps:
- name: 'gcr.io/cloud-builders/docker'
  args: ['tag', '-f', 'gcr.io/$PROJECT_ID/service:dev', 'gcr.io/$PROJECT_ID/service:prod']
images: ['gcr.io/$PROJECT_ID/service:prod']
}


# Once prod is ready, run the probes (built earlier) to check that things are
# working.
exec: test_prod(deploy_to_prod, build_probes) {
steps:
- name: 'gcr.io/$PROJECT_ID/probes'
  args: ['prod']
}
```
