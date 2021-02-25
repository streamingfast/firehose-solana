#!/bin/bash

TERRAFORM_BIN="${TERRAFORM_BIN:-terraform}"
PROJECT="dfuseio-global"
GCP_ACCESS_TOKEN="$(gcloud auth print-access-token --impersonate-service-account=serumviz@$PROJECT.iam.gserviceaccount.com)"

$TERRAFORM_BIN init -backend-config="access_token=$GCP_ACCESS_TOKEN"
$TERRAFORM_BIN workspace new "$PROJECT"
$TERRAFORM_BIN workspace select "$PROJECT"


