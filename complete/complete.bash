#!/bin/bash

# Complete a workflow execution.

GCS_PREFIX=$1
WORKFLOW_ID=$2
EXECUTION_ID=$3

echo "$ gsutil rsync /workflow_artifacts/out \"$GCS_PREFIX/$EXECUTION_ID/\""
gsutil rsync /workflow_artifacts/out "$GCS_PREFIX/$EXECUTION_ID/" || exit;

gcloud beta pubsub topics publish "workflow-$WORKFLOW_ID" "{\"completed\":\"$EXECUTION_ID\"}"
