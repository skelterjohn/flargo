# Complete a workflow execution.

GCS_PREFIX=$1
WORKFLOW_ID=$2
EXECUTION_ID=$3
shift; shift; shift
BLOCKS="$@"

gcloud beta pubsub subscriptions create "step-$EXECUTION_ID" --topic "workflow-$WORKFLOW_ID" || exit

gcloud beta pubsub subscriptions pull "step-$EXECUTION_ID"
