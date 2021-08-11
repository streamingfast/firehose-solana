#!/bin/bash

TERRAFORM_BIN="${TERRAFORM_BIN:-terraform}"
GCP_PROJECT="${GCP_PROJECT:-dfuseio-global}"
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
TFSTATE_FILE="$DIR/.terraform/terraform.tfstate"

if [ ! -f "$TFSTATE_FILE" ]; then
  echo "You must run init.sh at least once locally"
  exit 1
fi

GCP_ACCESS_TOKEN="$(gcloud auth print-access-token --impersonate-service-account=terraform@dfuseio-global.iam.gserviceaccount.com)"

# Refresh access token for backend first
cat <<< "$(jq ".backend.config.access_token = \"$GCP_ACCESS_TOKEN\"" < "$TFSTATE_FILE")" > "$TFSTATE_FILE"

$TERRAFORM_BIN workspace new "$GCP_PROJECT"
$TERRAFORM_BIN workspace select "$GCP_PROJECT"
TF_VAR_gcp_access_token=$GCP_ACCESS_TOKEN TF_VAR_gcp_project=$GCP_PROJECT $TERRAFORM_BIN $1
