locals {
  repo_root = abspath("${path.module}/../../..")
}

resource "terraform_data" "bootstrap_kind" {
  count = var.bootstrap_kind ? 1 : 0

  input = {
    repo_root = local.repo_root
  }

  provisioner "local-exec" {
    working_dir = self.input.repo_root
    command     = "powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/scripts/bootstrap-kind.ps1"
  }
}

resource "terraform_data" "k8s_apply" {
  depends_on = [terraform_data.bootstrap_kind]

  input = {
    manifest_dir    = var.manifest_dir
    kube_context    = var.kube_context
    kubeconfig_path = var.kubeconfig_path
    repo_root       = local.repo_root
  }

  triggers_replace = [
    var.manifest_dir,
    var.kube_context,
    var.kubeconfig_path,
    tostring(var.run_smoke_after_apply),
  ]

  provisioner "local-exec" {
    working_dir = self.input.repo_root
    environment = {
      KUBECONFIG = self.input.kubeconfig_path
    }
    command = "powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/terraform/scripts/use-context.ps1 -Context '${self.input.kube_context}'"
  }

  provisioner "local-exec" {
    working_dir = self.input.repo_root
    environment = {
      KUBECONFIG = self.input.kubeconfig_path
    }
    command = "powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/apply-lab.ps1"
  }

  provisioner "local-exec" {
    when        = destroy
    working_dir = self.input.repo_root
    environment = {
      KUBECONFIG = self.input.kubeconfig_path
    }
    command = "powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/terraform/scripts/destroy-k8s.ps1 -ManifestDir '${self.input.manifest_dir}' -Context '${self.input.kube_context}'"
  }
}

resource "terraform_data" "k8s_smoke" {
  count      = var.run_smoke_after_apply ? 1 : 0
  depends_on = [terraform_data.k8s_apply]

  input = {
    repo_root       = local.repo_root
    kubeconfig_path = var.kubeconfig_path
  }

  provisioner "local-exec" {
    working_dir = self.input.repo_root
    environment = {
      KUBECONFIG = self.input.kubeconfig_path
    }
    command = "powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/smoke-basic.ps1"
  }
}

