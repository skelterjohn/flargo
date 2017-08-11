#!/bin/bash

# Complete a workflow execution.

GCS_PREFIX=$1
WORKFLOW_ID=$2
EXECUTION_ID=$3

if ( stat out &> /dev/null ); then
  gsutil cp -r out "$GCS_PREFIX/$WORKFLOW_ID/$EXECUTION_ID/" || exit;
else
  echo "No output to copy."
fi

gcloud beta pubsub topics publish "workflow-$WORKFLOW_ID" "{\"completed\":\"$EXECUTION_ID\"}"
