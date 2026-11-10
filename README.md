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

**Challenge:** The original code has no backend, so Terraform keeps its state in a local `terraform.tfstate` file. In GitHub Actions every job runs on a new runner VM that is deleted when the job ends, so the state file disappears together with it. The next run has no state: it would try to create all resources again (and fail on existing ones), and `destroy` would not know what to delete. The state has to be shared between runners.

**Solution:** I moved the state to a remote S3 backend ([`backend.tf`](./infrastructure/backend.tf)). Every job runs `terraform init` against the same S3 bucket, so plan, apply and destroy always read and write the same state. Native S3 locking (`use_lockfile = true`) prevents two runs from changing it at the same time. The bucket is created once beforehand (see *One-time setup*, step 2), and its name is passed to `init` from the `TF_STATE_BUCKET` variable.

## Test results

| Test | Result |
|---|---|
| [PR checks + plan](https://github.com/melnykv2/week11/actions/runs/42813792293) | ✅ `Plan: 18 to add, 0 to change, 0 to destroy` (attempt 2) |
| [Manual apply](https://github.com/melnykv2/week11/actions/runs/42815517159) | ✅ 5 instances running. Both HTTP instances returned "Apache2 Ubuntu Default Page: It works" |
| [Manual destroy](https://github.com/melnykv2/week11/actions/runs/42816033088) | ✅ No resources left in AWS. The state contains 0 resources |
| [Push with a Terraform change](https://github.com/melnykv2/week11/actions/runs/42816756065) | ✅ Checks and plan only. Apply and destroy skipped |
