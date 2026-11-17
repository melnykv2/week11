# Terraform AWS Infrastructure with GitHub Actions CI/CD

![infra instance with multiple network](./img/03-multiple-network.png "infra instance with multiple network")

The Terraform code in [`infrastructure/`](./infrastructure) creates a VPC, 2 subnets, 2 HTTP instances (Apache) and 3 DB instances (MariaDB) in AWS (`eu-north-1`). The pipeline is in [`.github/workflows/terraform.yml`](./.github/workflows/terraform.yml).

> Task 1 (basic pipeline) is preserved at the [`task-1`](https://github.com/melnykv2/week11/tree/task-1) tag.

## Implementation

```
push / PR to main     ──► Init → Format → Validate → TFLint → Trivy → Plan
Run workflow: apply   ──► (same steps) → Apply → Terratest
Run workflow: destroy ──► Init → Destroy
```

All checks are steps of one job, so if any step fails, the steps after it are skipped. Pushes and PRs only check and plan; AWS changes only when the workflow is started manually.

- **TFLint** checks best practices with the `terraform` and AWS rulesets ([`.tflint.hcl`](./infrastructure/.tflint.hcl)).
- **Trivy** scans for misconfigurations and hardcoded secrets, and fails on HIGH and CRITICAL findings.
- **Terratest** ([`test/`](./test)) runs after apply and checks that:
  - all 5 EC2 instances (2 HTTP, 3 DB) are running
  - the VPC (`192.168.0.0/16`) and subnets (`192.168.1.0/24`, `192.168.2.0/24`) have the correct CIDRs
  - the DB instances have no public IP or Elastic IP, so the database is not reachable from the internet

## How to use

1. Create the S3 state bucket (`aws s3api create-bucket ...`) with versioning enabled.
2. In *Settings → Secrets and variables → Actions*, add the secrets `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`, and the variable `TF_STATE_BUCKET`.
3. Set `admin_cidr` in [`00-params.tf`](./infrastructure/00-params.tf) to the IP you SSH from (`x.x.x.x/32`).
4. Open a PR to run the checks and the plan. Use *Actions → Terraform CI/CD → Run workflow* with `apply` or `destroy` to deploy or delete.

## Challenges and solutions

| Challenge | Solution |
|---|---|
| Runners are deleted after each job, so local Terraform state was lost | Remote S3 backend with locking ([`backend.tf`](./infrastructure/backend.tf)) |
| TFLint: no version constraint for the AWS provider | Rule `terraform_required_providers` disabled, to keep the original provider configuration (the provider version stays unpinned) |
| `tflint --init` failed with `403 API rate limit exceeded` (runners share IPs) | The TFLint step uses the workflow's `GITHUB_TOKEN` |
| Trivy found 14 HIGH/CRITICAL issues | Fixed in code: IMDSv2 required, encrypted root disks, SSH only from `admin_cidr`. Egress to the internet is accepted in [`.trivyignore`](./.trivyignore), because the instances need it for `apt install` |

## Test results

| Run | Result |
|---|---|
| [Checks + plan](https://github.com/melnykv2/week11/actions/runs/44191035618) | ✅ TFLint, Trivy and plan passed |
| [Apply + Terratest](https://github.com/melnykv2/week11/actions/runs/44191409115) | ✅ All 3 Terratest checks passed; both HTTP instances returned "It works" |
| [Destroy](https://github.com/melnykv2/week11/actions/runs/44192327718) | ✅ No resources left in AWS |
| [Trivy gate](https://github.com/melnykv2/week11/actions/runs/43447179433) | ❌ Stopped at Trivy, plan skipped (proves the chain stops on failure) |
