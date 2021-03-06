/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package factory

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"golang.org/x/net/context"

	"k8s.io/apimachinery/pkg/runtime/schema"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	clientapi "k8s.io/client-go/pkg/api"
	clientapiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/kubernetes/pkg/storage/etcd/testing/testingcert"

	"github.com/coreos/etcd/integration"
	"github.com/coreos/etcd/pkg/transport"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestTLSConnection(t *testing.T) {
	scheme := runtime.NewScheme()
	codecs := runtimeserializer.NewCodecFactory(scheme)
	codec := codecs.LegacyCodec(schema.GroupVersion{Version: "v1"})

	// TODO: use k8s.io/apiserver internal type instead of borrowing it from client-go
	clientapi.AddToScheme(scheme)
	clientapiv1.AddToScheme(scheme)

	certFile, keyFile, caFile := configureTLSCerts(t)
	defer os.RemoveAll(filepath.Dir(certFile))

	tlsInfo := &transport.TLSInfo{
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   caFile,
	}

	cluster := integration.NewClusterV3(t, &integration.ClusterConfig{
		Size:      1,
		ClientTLS: tlsInfo,
	})
	defer cluster.Terminate(t)

	cfg := storagebackend.Config{
		Type:       storagebackend.StorageTypeETCD3,
		ServerList: []string{cluster.Members[0].GRPCAddr()},
		CertFile:   certFile,
		KeyFile:    keyFile,
		CAFile:     caFile,
		Codec:      codec,
	}
	storage, destroyFunc, err := newETCD3Storage(cfg)
	defer destroyFunc()
	if err != nil {
		t.Fatal(err)
	}
	err = storage.Create(context.TODO(), "/abc", &clientapi.Pod{}, nil, 0)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
}

func configureTLSCerts(t *testing.T) (certFile, keyFile, caFile string) {
	baseDir := os.TempDir()
	tempDir, err := ioutil.TempDir(baseDir, "etcd_certificates")
	if err != nil {
		t.Fatal(err)
	}
	certFile = path.Join(tempDir, "etcdcert.pem")
	if err := ioutil.WriteFile(certFile, []byte(testingcert.CertFileContent), 0644); err != nil {
		t.Fatal(err)
	}
	keyFile = path.Join(tempDir, "etcdkey.pem")
	if err := ioutil.WriteFile(keyFile, []byte(testingcert.KeyFileContent), 0644); err != nil {
		t.Fatal(err)
	}
	caFile = path.Join(tempDir, "ca.pem")
	if err := ioutil.WriteFile(caFile, []byte(testingcert.CAFileContent), 0644); err != nil {
		t.Fatal(err)
	}
	return certFile, keyFile, caFile
}
