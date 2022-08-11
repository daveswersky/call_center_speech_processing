cd /tf_deploy/project_setup
terraform init
export GCP_PROJECT=`gcloud config list --format 'value(core.project)' 2>/dev/null`
terraform plan -out project_setup.tfplan -var="project_id=$GCP_PROJECT"
terraform apply project_setup.tfplan
cd ..
terraform init
terraform plan -out callaudio.tfplan -var="project_id=$GCP_PROJECT" -var="service_account_email=transcription-project-sa@$GCP_PROJECT.iam.gserviceaccount.com"
terraform apply callaudio.tfplan