package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gocloud.dev/blob"

	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/memblob"
	_ "gocloud.dev/blob/s3blob"

	"github.com/meringu/terraform-bucket-registry/pkg/api"
)

var publishName string
var publishNamespace string
var publishVersion string
var publishDistDir string
var publishDestination string
var publishBucketURL string
var publishProviderProtocols []string
var publishGPGKeyID string
var publishGPGPublicKeyFile string

const discoveryPath = ".well-known/terraform.json"
const discoveryContent = `{
	"providers.v1": "/v1/providers/"
}
`

func init() {
	publishCmd.PersistentFlags().StringVar(&publishName, "name", "", "name of the Terraform provider to publish")
	publishCmd.PersistentFlags().StringVar(&publishNamespace, "namespace", "", "namespace of the Terraform provider to publish")
	publishCmd.PersistentFlags().StringVar(&publishVersion, "version", "", "version of the provider to publish")
	publishCmd.PersistentFlags().StringVar(&publishDistDir, "dist-dir", "./dist", "directory of the built provider to publish")
	publishCmd.PersistentFlags().StringVar(&publishDestination, "destination", "", "destination bucket to publish provider to")
	publishCmd.PersistentFlags().StringVar(&publishBucketURL, "bucket-url", "", "url for access to the bucket from Terraform")
	publishCmd.PersistentFlags().StringVar(&publishGPGKeyID, "gpg-key-id", "", "gpg key id")
	publishCmd.PersistentFlags().StringVar(&publishGPGPublicKeyFile, "gpg-public-key-file", "", "ascii armor public key")
	publishCmd.PersistentFlags().StringSliceVar(&publishProviderProtocols, "provider-protocol", []string{"4.0", "5.1"}, "supported provider protocols")

	publishCmd.MarkPersistentFlagRequired("name")
	publishCmd.MarkPersistentFlagRequired("namespace")
	publishCmd.MarkPersistentFlagRequired("version")
	publishCmd.MarkPersistentFlagRequired("dist-dir")
	publishCmd.MarkPersistentFlagRequired("destination")
	publishCmd.MarkPersistentFlagRequired("bucket-url")
	publishCmd.MarkPersistentFlagRequired("gpg-key-id")
	publishCmd.MarkPersistentFlagRequired("gpg-public-key-file")

	rootCmd.AddCommand(publishCmd)
}

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish a built Terraform provider to a bucket",
	Run: func(cmd *cobra.Command, args []string) {
		if err := publish(); err != nil {
			logrus.Fatal(err)
		}
	},
}

func publish() error {
	glob := filepath.Join(publishDistDir, fmt.Sprintf("terraform-provider-%s_%s_*.zip", publishName, publishVersion))
	paths, err := filepath.Glob(glob)
	if err != nil {
		return err
	}

	if len(paths) == 0 {
		return fmt.Errorf("No files found for matching %s", glob)
	}

	shasumsLocal := fmt.Sprintf("%s/terraform-provider-%s_%s_SHA256SUMS", publishDistDir, publishName, publishVersion)
	shasumsPath := fmt.Sprintf("v1/providers/%s/%s/%s/terraform-provider-%s_%s_SHA256SUMS", publishNamespace, publishName, publishVersion, publishName, publishVersion)
	shasumsSigPath := fmt.Sprintf("%s.sig", shasumsPath)
	shasumsSigLocal := fmt.Sprintf("%s.sig", shasumsLocal)

	ctx := context.Background()

	bucket, err := blob.OpenBucket(ctx, publishDestination)
	if err != nil {
		return err
	}
	defer bucket.Close()

	// Write discovery
	logrus.Infof("Uploading %s", discoveryPath)
	bucket.WriteAll(ctx, discoveryPath, []byte(discoveryContent), &blob.WriterOptions{ContentType: "application/json"})

	pubkey, err := os.ReadFile(publishGPGPublicKeyFile)
	if err != nil {
		return err
	}

	// Upload each package
	for _, path := range paths {
		base := filepath.Base(path)
		suffix := strings.TrimPrefix(base, fmt.Sprintf("terraform-provider-%s_%s_", publishName, publishVersion))
		osArch := strings.TrimSuffix(suffix, ".zip")
		components := strings.Split(osArch, "_")
		if len(components) != 2 {
			continue
		}

		publishOS := components[0]
		publishArch := components[1]

		downloadPath := fmt.Sprintf("v1/providers/%s/%s/%s/download/%s", publishNamespace, publishName, publishVersion, base)

		// Write zip
		logrus.Infof("Uploading %s", downloadPath)
		err := writeFile(ctx, bucket, downloadPath, path, "application/zip")
		if err != nil {
			return err
		}

		shasum, err := shasumFor(path, shasumsLocal)
		if err != nil {
			return err
		}

		// write manifest
		providerPackage := &api.ProviderPackage{
			Protocols:           publishProviderProtocols,
			OS:                  publishOS,
			Arch:                publishArch,
			Filename:            base,
			DownloadURL:         fmt.Sprintf("%s/%s", publishBucketURL, downloadPath),
			ShasumsURL:          fmt.Sprintf("%s/%s", publishBucketURL, shasumsPath),
			ShasumsSignatureURL: fmt.Sprintf("%s/%s", publishBucketURL, shasumsSigPath),
			Shasum:              shasum,
			SigningKeys: &api.SigningKeys{
				GPGPublicKeys: []*api.GPGPublicKey{
					&api.GPGPublicKey{
						KeyID:          publishGPGKeyID,
						ASCIIArmor:     string(pubkey),
						TrustSignature: "",
						Source:         "",
						SourceURL:      "",
					},
				},
			},
		}
		b, err := json.MarshalIndent(providerPackage, "", "  ")
		if err != nil {
			return err
		}
		packagePath := fmt.Sprintf("v1/providers/%s/%s/%s/download/%s/%s", publishNamespace, publishName, publishVersion, publishOS, publishArch)
		logrus.Infof("Uploading %s", packagePath)
		err = bucket.WriteAll(ctx, packagePath, b, &blob.WriterOptions{ContentType: "application/json"})
		if err != nil {
			return err
		}
	}

	// Upload checksum
	logrus.Infof("Uploading %s", shasumsPath)
	err = writeFile(ctx, bucket, shasumsPath, shasumsLocal, "text/plain")
	if err != nil {
		return err
	}

	// Upload checksum signature
	logrus.Infof("Uploading %s", shasumsSigPath)
	err = writeFile(ctx, bucket, shasumsSigPath, shasumsSigLocal, "application/octet-stream")
	if err != nil {
		return err
	}

	// Index all packages for the provider
	iter := bucket.List(&blob.ListOptions{
		Prefix:    fmt.Sprintf("v1/providers/%s/%s/", publishNamespace, publishName),
		Delimiter: "/",
	})

	providerVersions := &api.ProviderVersions{}

	// Loop through versions
	for {
		obj, err := iter.Next(ctx)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		version := strings.Split(obj.Key, "/")[4]

		verIter := bucket.List(&blob.ListOptions{
			Prefix:    fmt.Sprintf("v1/providers/%s/%s/%s/download/", publishNamespace, publishName, version),
			Delimiter: "/",
		})

		platforms := []*api.Platform{}

		var versionProtocols []string

		// Loop through os for version
		for {
			obj, err := verIter.Next(ctx)
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}

			os := strings.Split(obj.Key, "/")[6]

			osIter := bucket.List(&blob.ListOptions{
				Prefix:    fmt.Sprintf("v1/providers/%s/%s/%s/download/%s/", publishNamespace, publishName, version, os),
				Delimiter: "/",
			})

			// Loop through arch for os version
			for {
				obj, err := osIter.Next(ctx)
				if err != nil {
					if err == io.EOF {
						break
					}
					return err
				}

				arch := strings.Split(obj.Key, "/")[7]

				platforms = append(platforms, &api.Platform{
					OS:   os,
					Arch: arch,
				})

				if len(versionProtocols) == 0 {
					content, err := bucket.ReadAll(ctx, fmt.Sprintf("v1/providers/%s/%s/%s/download/%s/%s", publishNamespace, publishName, version, os, arch))
					if err != nil {
						return err
					}

					pkg := &api.ProviderPackage{}
					err = json.Unmarshal(content, pkg)
					if err != nil {
						return err
					}

					versionProtocols = pkg.Protocols
				}
			}
		}

		if len(platforms) > 0 {
			providerVersions.Versions = append(providerVersions.Versions, &api.ProviderVersion{
				Version:   version,
				Protocols: versionProtocols,
				Platforms: platforms,
			})
		}
	}

	b, err := json.MarshalIndent(providerVersions, "", "  ")
	if err != nil {
		return err
	}
	pvPath := fmt.Sprintf("v1/providers/%s/%s/versions", publishNamespace, publishName)
	logrus.Infof("Uploading %s", pvPath)
	err = bucket.WriteAll(ctx, pvPath, b, &blob.WriterOptions{ContentType: "application/json"})
	if err != nil {
		return err
	}

	return nil
}

func writeFile(ctx context.Context, bucket *blob.Bucket, path string, source string, contentType string) error {
	w, err := bucket.NewWriter(ctx, path, nil)
	if err != nil {
		return err
	}
	defer w.Close()
	r, err := os.Open(source)
	if err != nil {
		return err
	}
	defer r.Close()
	_, err = io.Copy(w, r)
	return err
}

func shasumFor(file, shasumsFile string) (string, error) {
	content, err := os.ReadFile(shasumsFile)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		components := strings.Split(line, "  ")
		if len(components) != 2 {
			return "", fmt.Errorf("Couldn't parse %s", shasumsFile)
		}

		if components[1] == filepath.Base(file) {
			return components[0], nil
		}
	}
	return "", fmt.Errorf("Couldn't match shasum for %s", file)
}
