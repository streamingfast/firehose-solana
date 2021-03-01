#!/bin/bash

TERRAFORM_BIN="${TERRAFORM_BIN:-terraform}"
GCP_PROJECT="${GCP_PROJECT:-dfuseio-global}"
GCP_ACCESS_TOKEN="$(gcloud auth print-access-token --impersonate-service-account=terraform@dfuseio-global.iam.gserviceaccount.com)"

$TERRAFORM_BIN workspace new "$GCP_PROJECT"
$TERRAFORM_BIN workspace select "$GCP_PROJECT"
TF_VAR_gcp_access_token=$GCP_ACCESS_TOKEN TF_VAR_gcp_project=$GCP_PROJECT $TERRAFORM_BIN init
