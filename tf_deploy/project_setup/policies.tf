module "compute-vm-external-ip-access" {
  source            = "terraform-google-modules/org-policy/google"
  version           = "~> 3.0.2"
  policy_for        = "project"
  project_id        = var.project_id
  constraint        = "constraints/compute.vmExternalIpAccess"
  policy_type       = "list"
  enforce           = "false"
}

module "compute-vm-required-shielded"{
  source            = "terraform-google-modules/org-policy/google"
  version           = "~> 3.0.2"
  policy_for        = "project"
  project_id        = var.project_id
  constraint        = "constraints/compute.requireShieldedVm"
  policy_type       = "boolean"
  enforce           = "false"
}

module "functions-allowed-ingress"{
  source            = "terraform-google-modules/org-policy/google"
  version           = "~> 3.0.2"
  policy_for        = "project"
  project_id        = var.project_id
  constraint        = "constraints/cloudfunctions.allowedIngressSettings"
  policy_type       = "list"
  enforce           = "false"
}

module "uniform-bucket-access"{
  source            = "terraform-google-modules/org-policy/google"
  version           = "~> 3.0.2"
  policy_for        = "project"
  project_id        = var.project_id
  constraint        = "constraints/storage.uniformBucketLevelAccess"
  policy_type       = "boolean"
  enforce           = "false"
}