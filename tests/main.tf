# Example Terraform configuration for testing multi-profile functionality

terraform {
  required_version = ">= 1.0"
}

# Example variable
variable "environment" {
  description = "Environment name"
  type        = string
}

variable "region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

# Example resource - using local_file for simplicity (no AWS credentials needed)
resource "local_file" "example" {
  content  = "Hello from ${var.environment} environment in ${var.region}!"
  filename = "${path.module}/output-${var.environment}.txt"
}

# Output
output "message" {
  value = "Successfully deployed to ${var.environment} environment"
} 

