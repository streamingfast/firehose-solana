#!/bin/bash

TERRAFORM_BIN="${TERRAFORM_BIN:-terraform}"
GCP_PROJECT="${GCP_PROJECT:-dfuseio-global}"
GCP_ACCESS_TOKEN="$(gcloud auth print-access-token --impersonate-service-account=terraform@$GCP_PROJECT.iam.gserviceaccount.com)"

$TERRAFORM_BIN init -backend-config="access_token=$GCP_ACCESS_TOKEN"
$TERRAFORM_BIN workspace new "$GCP_PROJECT"
$TERRAFORM_BIN workspace select "$GCP_PROJECT"


