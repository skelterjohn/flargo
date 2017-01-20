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
