# Terraform Bucket Registry

This project implements the Terraform Provider Registry Protocol on top of common bucket/blob storage used by Cloud Providers

See https://www.terraform.io/docs/internals/provider-registry-protocol.html for more details about the protocol.

## Use case

- You have created a Terraform provider(s).
- Your Terraform providers aren't fit to share on the public Terraform Registry.
- You want to distribute your Terraform providers with semantic versioning support.
- You want to manage the installation of your Terraform providers during `terraform init`.
- You have access to AWS S3, Azure blob, GCP Cloud Storage, or similar APIs for hosting static websites.

## Installation

### MacOS

```
brew tap meringu/terraform-bucket-registry
brew install terraform-bucket-registry
```

### Releases

Download the latest release from https://github.com/meringu/terraform-bucket-registry/releases

## Usage

1. Create a bucket website. For example let's pretend we have provisioned an AWS S3 bucket called `my-terraform-registry` to serve content at `https://my-terraform-registry.com`

1. Create a GPG key for signing. You may also use an existing key.

    ```
    gpg --gen-key
    ```

1. Export the GPG public key

    ```
    gpg --armor --export --output key.asc firstname.lastname@example.com
    ```

2. Build your Terraform provider. The following [GoReleaser](https://github.com/goreleaser/goreleaser) config can be used as a starting point. This will build the provider into the `dist/` folder.

    ```yaml
    ---
    builds:
    - env:
      - CGO_ENABLED=0
    mod_timestamp: "{{ .CommitTimestamp }}"
      flags:
        - -trimpath
      ldflags:
        - "-s -w -X main.version={{.Version}} -X main.commit={{.Commit}}"
      goos:
        - freebsd
        - windows
        - linux
        - darwin
      goarch:
        - amd64
        - "386"
        - arm
        - arm64
      ignore:
        - goos: darwin
          goarch: "386"
      binary: "{{ .ProjectName }}_v{{ .Version }}"
      archives:
        - format: zip
          name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
      checksum:
      name_template: "{{ .ProjectName }}_{{ .Version }}_SHA256SUMS"
      algorithm: sha256
      signs:
        - artifacts: checksum
          args:
            - "--batch"
            - "--local-user"
            - "{{ .Env.GPG_FINGERPRINT }}"
            - "--output"
            - "${signature}"
            - "--detach-sign"
            - "${artifact}"
    ```

1. Release the provider into the bucket registry.

    ```bash
    terraform-bucket-registry publish \
        --name $provider_name \
        --namespace $provider_namespace \
        --dist-dir ./dist/ \
        --version $provider_version \
        --destination s3://my-terraform-registry \
        --bucket-url https://my-terraform-registry.com \
        --gpg-key-id $gpg_key_id \
        --gpg-public-key-file key.asc
    ```

## Example

There is an end-to-end example implemented with `docker-compose`.

This example does the following:
- Build an example Terraform Provider
- Create a GPG key
- Use GoReleaser to build, package, and sign the provider
- Create SSL certificates
- Run the terraform-bucket-registry server
- Publish the provider to the registry
- Init and apply example Terraform using the terraform-bucket-registry server

```
docker-compose -f examples/terraform-provider-example/docker-compose.yaml build
docker-compose -f examples/terraform-provider-example/docker-compose.yaml up
```

Until docker-compose can represent dependencies on containers finishing you may need to run the up command twice.
