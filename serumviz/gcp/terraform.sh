#!/bin/bash

TERRAFORM_BIN="${TERRAFORM_BIN:-terraform}"
PROJECT="dfuseio-global"
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
TFSTATE_FILE="$DIR/.terraform/terraform.tfstate"

if [ ! -f "$TFSTATE_FILE" ]; then
  echo "You must run init.sh at least once locally"
  exit 1
fi

GCP_ACCESS_TOKEN="$(gcloud auth print-access-token --impersonate-service-account=terraform@$PROJECT.iam.gserviceaccount.com)"

# Refresh access token for backend first
cat <<< "$(jq ".backend.config.access_token = \"$GCP_ACCESS_TOKEN\"" < "$TFSTATE_FILE")" > "$TFSTATE_FILE"

TF_VAR_gcp_access_token=$GCP_ACCESS_TOKEN TF_VAR_gcp_project=$PROJECT $TERRAFORM_BIN $1
