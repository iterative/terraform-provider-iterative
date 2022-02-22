# Amazon Machine Images

## Updating Images

1. Edit `setup.sh` and open a pull request with the changes.
2. Run `packer build .` in this directory and wait for the operation to complete.

This operation needs to be run from the [`dvc-cml-terraform-provider`](https://github.com/iterative/itops/blob/e2423bf5b253896c68432a7e20d186918ed00703/cml/terraform/cml-terraform-provider.tf#L1-L3) IAM user, so `packer` can assume the [`cml-packer`](https://github.com/iterative/itops/blob/e2423bf5b253896c68432a7e20d186918ed00703/cml/terraform/packer-role.tf) role.
