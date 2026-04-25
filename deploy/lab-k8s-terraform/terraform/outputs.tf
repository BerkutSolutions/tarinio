output "manifest_dir" {
  value       = var.manifest_dir
  description = "Applied manifest directory."
}

output "kube_context" {
  value       = var.kube_context
  description = "Kubernetes context used by Terraform orchestration."
}
