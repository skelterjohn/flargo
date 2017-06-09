# coord

The `coord` container image coordinates a `flargo` workflow.

It listens to a Cloud Pub/Sub topic to keep track of each `flargo` execution, such that only the id of the build running the coord is needed in order to get a view of the entire workflow.

For each execution that begins, ends, is retried or skipped, the `coord` build step will write a log message that can be consulted by the `flargo` tool later.

For this container image to run as a Container Builder step, the builder service account needs following permissions:
 - pubsub.subscriptions.consume
 - pubsub.subscriptions.create
 - pubsub.topics.attachSubscription
 - pubsub.topics.create
 - pubsub.topics.publish
