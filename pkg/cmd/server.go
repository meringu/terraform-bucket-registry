package cmd

import (
	"context"
	"io"
	"log"
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gocloud.dev/blob"
	"gocloud.dev/gcerrors"

	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/memblob"
	_ "gocloud.dev/blob/s3blob"
)

var serverBucketURL string
var serverTLSPublicKey string
var serverTLSPrivateKey string

func init() {
	serverCmd.PersistentFlags().StringVar(&serverBucketURL, "bucket-url", "", "bucket to serve")
	serverCmd.PersistentFlags().StringVar(&serverTLSPublicKey, "tls-public-key", "", "path to TLS public key")
	serverCmd.PersistentFlags().StringVar(&serverTLSPrivateKey, "tls-private-key", "", "public to TLS private key")

	serverCmd.MarkPersistentFlagRequired("bucket-url")

	rootCmd.AddCommand(serverCmd)
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Serve a terraform registry from a bucket",
	Run: func(cmd *cobra.Command, args []string) {
		s, err := NewServer()
		if err != nil {
			log.Fatal(err)
		}
		defer s.cleanup()
		if serverTLSPublicKey != "" {
			err := http.ListenAndServeTLS(":443", serverTLSPublicKey, serverTLSPrivateKey, s)
			if err != nil {
				log.Fatal("ListenAndServe: ", err)
			}
		} else {
			err := http.ListenAndServe(":80", s)
			if err != nil {
				log.Fatal("ListenAndServe: ", err)
			}
		}
	},
}

type server struct {
	bucket *blob.Bucket
}

func NewServer() (*server, error) {
	ctx := context.Background()

	bucket, err := blob.OpenBucket(ctx, serverBucketURL)
	if err != nil {
		return nil, err
	}

	return &server{bucket: bucket}, nil
}

func (s *server) cleanup() {
	s.bucket.Close()
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reader, err := s.bucket.NewReader(r.Context(), r.RequestURI, nil)
	if err != nil {
		if gcerrors.Code(err) == gcerrors.NotFound {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not Found"))
			logrus.Info("404 - %s", r.RequestURI)
			return
		}
		logrus.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		logrus.Info("500 - %s", r.RequestURI)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", reader.ContentType())
	w.WriteHeader(http.StatusOK)

	_, err = io.Copy(w, reader)
	if err != nil {
		logrus.Error(err)
	} else {
		logrus.Info("200 - %s", r.RequestURI)
	}
}
