# Terraform AWS Infrastructure with GitHub Actions CI/CD

![infra instance with multiple network](./img/03-multiple-network.png "infra instance with multiple network")

The Terraform code in [`infrastructure/`](./infrastructure) creates in AWS (`eu-north-1`):
-   1 VPC
-   2 subnets (http, db)
-   2 HTTP instances (Apache, with Elastic IPs)
-   3 DB instances (MariaDB)

The whole lifecycle is managed by a GitHub Actions pipeline: [`.github/workflows/terraform.yml`](./.github/workflows/terraform.yml).

## Implementation

```
push / PR to main     ──► Format & Validate ──► Plan
Run workflow: apply   ──► Format & Validate ──► Plan ──► Apply
Run workflow: destroy ──► Format & Validate ──► Destroy
```

| Job | What it does |
|---|---|
| **Format & Validate** | `terraform fmt -check`, `terraform init -backend=false`, `terraform validate`. Uses no AWS credentials. |
| **Plan** | `terraform plan -out=tfplan`. Shows the plan on the run's Summary page and uploads `tfplan` and `.terraform.lock.hcl` as an artifact. |
| **Apply (manual)** | Downloads the saved plan and runs `terraform apply tfplan`, so exactly the reviewed plan is applied. |
| **Destroy (manual)** | `terraform destroy -auto-approve`. |

Key points:
- **Automatic runs (push or PR) never change AWS.** Apply and destroy run only when started manually.
- **Remote state in S3** ([`infrastructure/backend.tf`](./infrastructure/backend.tf)), with native S3 locking. The bucket name is passed at `init` from a GitHub variable.
- **One run at a time** (`concurrency`), and a running apply is never cancelled.
- **The Terraform version is pinned** (`1.16.5`). AWS keys are available only to the jobs that need them.

## How to use

**One-time setup**
1. Create an IAM user with EC2 and S3 permissions, and an access key for it.
2. Create the S3 state bucket. It must exist before the first `terraform init`:
   ```bash
   aws s3api create-bucket --bucket <bucket> --region eu-north-1 \
     --create-bucket-configuration LocationConstraint=eu-north-1
   aws s3api put-bucket-versioning --bucket <bucket> --versioning-configuration Status=Enabled
   ```
3. In GitHub, go to *Settings → Secrets and variables → Actions* and add:
   - **Secrets:** `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`
   - **Variables:** `TF_STATE_BUCKET` (the bucket name), `AWS_REGION` (`eu-north-1`)

**Daily use**
- **Check a change:** open a PR or push to `main`. Format & Validate and Plan run automatically. The plan is on the run's **Summary** page.
- **Deploy:** *Actions → Terraform CI → Run workflow* → branch `main`, action `apply`.
- **Delete everything:** *Actions → Terraform CI → Run workflow* → action `destroy`.

**Locally**
```bash
cd infrastructure
terraform init -backend-config="bucket=<bucket>"   # same state as the pipeline
terraform plan
```

## Challenges and solutions

| Challenge | Solution |
|---|---|
| **State was lost between runs.** Every GitHub job runs on a new VM, so local state disappeared after apply, and destroy would not know what to delete. | Added an S3 backend (`backend.tf`). Fixed settings are in the file. The bucket comes from the `TF_STATE_BUCKET` variable, because backend blocks can't use Terraform variables. |
| **`terraform init` did not create the bucket.** The backend is set up before Terraform reads any resources, so the bucket can't be created by the same configuration. | Created the bucket once with the AWS CLI, with versioning on. |
| **The original code failed `fmt -check`.** `00-params.tf` was misaligned, and fixing it by hand moved the error instead of fixing it. | Ran `terraform fmt -recursive`, which changed whitespace only. |
| **Plan failed with `The value cannot be empty`.** The log showed `bucket=`, because `TF_STATE_BUCKET` had not been created as a Variable (only the Secrets existed). GitHub replaces a missing variable with an empty string. | Added the repository Variables, then *Re-run failed jobs*. |
| **Plan and apply run on different VMs**, so they could use different provider versions, and `upload-artifact` skips hidden files such as `.terraform.lock.hcl`. | The lock file is uploaded together with `tfplan` (`include-hidden-files: true`). |
| **PRs from forks can't read secrets**, so Plan would always fail for them. | Plan is skipped for fork PRs. Only the credential-free checks run. |
| **"Run workflow" redirected to a 404 page.** | The run had actually started. The Actions page was out of date after the merge, and refreshing it before clicking fixes it. |

## Test results

| Test | Result |
|---|---|
| [PR checks + plan](https://github.com/melnykv2/week11/actions/runs/42813792293) | ✅ `Plan: 18 to add, 0 to change, 0 to destroy` (attempt 2) |
| [Manual apply](https://github.com/melnykv2/week11/actions/runs/42815517159) | ✅ 5 instances running. Both HTTP instances returned "Apache2 Ubuntu Default Page: It works" |
| [Manual destroy](https://github.com/melnykv2/week11/actions/runs/42816033088) | ✅ No resources left in AWS. The state contains 0 resources |
| [Push with a Terraform change](https://github.com/melnykv2/week11/actions/runs/42816756065) | ✅ Checks and plan only. Apply and destroy skipped |
