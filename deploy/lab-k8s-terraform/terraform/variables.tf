variable "bootstrap_kind" {
  description = "When true, run kind bootstrap before k8s apply."
  type        = bool
  default     = true
}

variable "kubeconfig_path" {
  description = "Path to kubeconfig used by kubectl."
  type        = string
  default     = ""
}

variable "kube_context" {
  description = "Kubernetes context to use. Empty means current context."
  type        = string
  default     = "kind-tarinio-lab"
}

variable "manifest_dir" {
  description = "Path to kustomize manifests directory."
  type        = string
  default     = "../k8s/manifests"
}

variable "run_smoke_after_apply" {
  description = "Run basic smoke script after apply."
  type        = bool
  default     = true
}
