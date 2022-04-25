---
page_title: Getting Started
---

# Getting Started

## Requirements

- [Install Terraform 1.0+](https://learn.hashicorp.com/tutorials/terraform/install-cli#install-terraform), e.g.:

  - Brew (Homebrew/Mac OS): `brew tap hashicorp/tap && brew install hashicorp/tap/terraform`
  - Choco (Chocolatey/Windows): `choco install terraform`
  - Conda (Anaconda): `conda install -c conda-forge terraform`
  - Debian (Ubuntu/Linux):

    ```console
    sudo apt-get update && sudo apt-get install -y gnupg software-properties-common curl
    curl -fsSL https://apt.releases.hashicorp.com/gpg | sudo apt-key add -
    sudo apt-add-repository "deb [arch=amd64] https://apt.releases.hashicorp.com $(lsb_release -cs) main"
    sudo apt-get update && sudo apt-get install terraform
    ```

- Create an account with any supported cloud vendor and expose its [authentication credentials via environment variables][auth]

[auth]: https://registry.terraform.io/providers/iterative/iterative/latest/docs/guides/authentication

## Define a Task

In a project root directory, create a file named `main.tf` with the following contents:

```hcl
terraform {
  required_providers { iterative = { source = "iterative/iterative" } }
}
provider "iterative" {}

resource "iterative_task" "example" {
  cloud      = "aws" # or any of: gcp, az, k8s
  machine    = "m"   # medium. Or any of: l, xl, m+k80, xl+v100, ...
  spot        = 0    # auto-price. Default -1 to disable, or >0 for hourly USD limit
  disk_size  = 30    # GB

  storage {
    workdir = "."       # default blank (don't upload)
    output  = "results" # default blank (don't upload). Relative to workdir
  }
  script = <<-END
    #!/bin/bash

    # create output directory if needed
    mkdir -p results
    # read last result (in case of spot/preemptible instance recovery)
    if test -f results/epoch.txt; then EPOCH="$(cat results/epoch.txt)"; fi
    EPOCH=$${EPOCH:-1}  # start from 1 if last result not found

    echo "(re)starting training loop from $EPOCH up to 1337 epochs"
    for epoch in $(seq $EPOCH 1337); do
      sleep 1
      echo "$epoch" | tee results/epoch.txt
    done
  END
}
```

See [the reference](https://registry.terraform.io/providers/iterative/iterative/latest/docs/resources/task#argument-reference) for the full list of options for `main.tf` -- including more information on [`machine` types](https://registry.terraform.io/providers/iterative/iterative/latest/docs/resources/task#machine-type) with and without GPUs.

-> **Note:** The `script` argument must begin with a valid [shebang](<https://en.wikipedia.org/wiki/Shebang_(Unix)>), and can take the form of a [heredoc string](https://www.terraform.io/docs/language/expressions/strings.html#heredoc-strings) or [a `file()` function](https://www.terraform.io/docs/language/functions/file.html) function (e.g. `file("task_run.sh")`).

The project layout should look similar to this:

```
project/
├── main.tf
└── results/
    └── epoch.txt (created in the cloud and downloaded locally)
```

## Initialise Terraform

```console
$ terraform init
```

This command will check `main.tf` and download the required TPI plugin.

~> **Warning:** None of the subsequent commands will work without first setting some [authentication environment variables][auth].

## Run Task

```console
$ TF_LOG_PROVIDER=INFO terraform apply
```

This command will:

1. Create all the required cloud resources (provisioning a `machine` with `disk_size` storage).
2. Upload the working directory (`workdir`) to the cloud.
3. Launch the task `script`.
4. Terminate the `machine` on `script` completion/error.

With spot/preemptible instances (`spot >= 0`), auto-recovery logic and persistent (`disk_size`) storage will be used to relaunch interrupted tasks.

-> **Note:** A large `workdir` may take a long time to upload.

~> **Warning:** To take full advantage of spot instance recovery, a `script` should start by cheching the disk for results (recovered from a previous interrupted run).

-> **Note:** The [`id`](https://registry.terraform.io/providers/iterative/iterative/latest/docs/resources/task#id) returned by `terraform apply` (i.e. `[id=tpi-···]`) can be used to locate the created cloud resources through the cloud's web console or command–line tool.

## Query Status

```console
$ TF_LOG_PROVIDER=INFO terraform refresh
$ TF_LOG_PROVIDER=INFO terraform show
```

These commands will:

1. Query the task status from the cloud.
2. Display the task status.

## End Task

```console
$ TF_LOG_PROVIDER=INFO terraform destroy
```

This command will:

1. Download the `output` directory from the cloud.
2. Delete all the cloud resources created by `terraform apply` (terminating `machine` if it's still running and removing the persistent `disk_size` storage).

In this example, after running `terraform destroy`, the `results` directory should contain a file named `epoch.txt` with the text `1337`.

-> **Note:** A large `output` directory may take a long time to download.

## Debugging

Use `TF_LOG_PROVIDER=DEBUG` in lieu of `INFO` to increase verbosity for debugging. See the [logging docs](https://www.terraform.io/plugin/log/managing) for a full list of options.

In case of errors within the `script` itself, both `stdout` and `stderr` are available from the [status](#query-status).

Advanced users may also want to SSH to debug failed scripts. This means preventing TPI from terminating the instance on `script` errors. For example:

```hcl
timeout     = 60*60*24               # 24h
environment = { GITHUB_ACTOR = "" }  # optional: GitHub username
script      = <<-END
  #!/bin/bash
  trap 'echo script error: waiting 24h for debugging over SSH. Run \"terraform destroy\" to stop waiting; sleep 24h' ERR
  # optional: allow GitHub user's ssh keys.
  # alternatively, use `ssh_private_key` and `addresses` from
  # https://registry.terraform.io/providers/iterative/iterative/latest/docs/resources/task#attribute-reference
  curl "https://github.com/$GITHUB_ACTOR.keys" >> "$HOME/.ssh/authorized_keys"

  # ... rest of script ...
END
```
