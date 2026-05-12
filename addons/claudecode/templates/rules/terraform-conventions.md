# Terraform/OpenTofu Conventions

- Use modules to organize infrastructure. Keep root modules thin.
- Pin provider and module versions explicitly. Use `~>` for minor version constraints.
- Name resources with `snake_case`. Use descriptive names reflecting purpose.
- Always run `terraform plan` and review before applying changes.
- Use `terraform fmt` for consistent formatting before every commit.
- Store remote state with locking (S3+DynamoDB, GCS, Terraform Cloud). Never commit `.tfstate`.
- Use variables and locals for values referenced more than once.
- Define output values for attributes that downstream modules need.
- Use `terraform validate` and `tflint` in CI.
- Tag all resources with `project`, `environment`, and `managed-by = "terraform"`.
- Separate environments using workspaces or separate root modules — not conditionals.
