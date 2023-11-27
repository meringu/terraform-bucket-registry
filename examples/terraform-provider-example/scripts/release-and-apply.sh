#!/bin/bash -ex

VERSION=0.0.1

cd "$(dirname "$0")/.."

# Cleanup
rm -rf .git/ dist/

# GPG
apk add --no-cache gnupg

if ! gpg --armor --list-keys --batch you@example.com; then
    cat | gpg --batch --gen-key <<EOF
    Key-Type: RSA
    Key-Length: 1024
    Name-Real: Your Name
    Name-Comment: with stupid passphrase
    Name-Email: you@example.com
    Expire-Date: 0
    %no-protection
    %commit
EOF
fi

export GPG_FINGERPRINT=$(gpg --with-colons --fingerprint | awk -F: '$1 == "fpr" {print $10;}' | head -n 1)
export GPG_KEY_ID=$(gpg --list-keys --keyid-format long --batch --with-colons you@example.com | grep pub | cut -d ':' -f 5)
export GPG_PUBLIC_KEY_FILE="$(pwd)/public.pgp"

rm -f $GPG_PUBLIC_KEY_FILE
gpg --output $GPG_PUBLIC_KEY_FILE --armor --export --batch


# Prep repo
git init
git branch -m main
git config user.email "you@example.com"
git config user.name "Your Name"
git remote add origin git@github.com:foo/terraform-provider-example.git
git add -A
git commit -am "init"
git tag v$VERSION

export GITHUB_TOKEN=foobar

goreleaser

pushd ../..
go mod download
go build -o /usr/local/bin/terraform-bucket-registry main.go
terraform-bucket-registry publish \
    --name example \
    --namespace foo \
    --dist-dir ./examples/terraform-provider-example/dist/ \
    --version $VERSION \
    --destination file:///var/lib/bucket \
    --bucket-url https://registry \
    --gpg-key-id $GPG_KEY_ID \
    --gpg-public-key-file $GPG_PUBLIC_KEY_FILE
popd

if ! command -v terraform > /dev/null; then
    curl https://releases.hashicorp.com/terraform/1.5.7/terraform_1.5.7_linux_amd64.zip > /tmp/terraform.zip
    unzip /tmp/terraform.zip
    mv terraform /usr/local/bin/terraform
fi

cat /etc/ssl/registry/ca.pem > /etc/ssl/certs/registry.pem

# clean terraform run
rm -rf examples/basic/.terraform
rm -f examples/basic/.terraform.lock.hcl
terraform -chdir=examples/basic init
terraform -chdir=examples/basic apply -auto-approve
