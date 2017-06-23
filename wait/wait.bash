# Complete a workflow execution.

GCS_PREFIX=$1
WORKFLOW_ID=$2
SUBSCRIPTION=$3
shift; shift; shift
BLOCKS="$@"

gcloud beta pubsub subscriptions pull "$SUBSCRIPTION"
