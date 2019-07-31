// Copyright 2018 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package main

import (
	"io/ioutil"
	"net/http"

	"github.com/drone/drone-go/plugin/secret"
	"github.com/drone/drone-kubernetes-secrets/plugin"
	"github.com/drone/drone-kubernetes-secrets/server"

	"github.com/ericchiang/k8s"
	"github.com/ghodss/yaml"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"

	_ "github.com/joho/godotenv/autoload"
)

type config struct {
	Debug     bool   `envconfig:"DEBUG"`
	Address   string `envconfig:"SERVER_ADDRESS"`
	Secret    string `envconfig:"SECRET_KEY"`
	Config    string `envconfig:"KUBERNETES_CONFIG"`
	Namespace string `envconfig:"KUBERNETES_NAMESPACE"`
}

func main() {
	spec := new(config)
	err := envconfig.Process("", spec)
	if err != nil {
		logrus.Fatal(err)
	}

	if spec.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if spec.Secret == "" {
		logrus.Fatalln("missing secret key")
	}
	if spec.Address == "" {
		spec.Address = ":3000"
	}
	if spec.Namespace == "" {
		spec.Namespace = "default"
	}

	client, err := createClient(spec.Config)
	if err != nil {
		logrus.Fatal(err)
	}

	handler := secret.Handler(
		spec.Secret,
		plugin.New(client, spec.Namespace),
		logrus.StandardLogger(),
	)
	healthzHandler := server.HandleHealthz()

	logrus.Infof("server listening on address %s", spec.Address)

	http.Handle("/", handler)
	http.Handle("/healthz", healthzHandler)
	logrus.Fatal(http.ListenAndServe(spec.Address, nil))
}

func createClient(path string) (*k8s.Client, error) {
	if path == "" {
		return k8s.NewInClusterClient()
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config k8s.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return k8s.NewClient(&config)
}
