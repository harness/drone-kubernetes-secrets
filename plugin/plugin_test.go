// Copyright 2018 Drone.IO Inc
// Use of this software is governed by the Business Source License
// that can be found in the LICENSE file.

package plugin

import (
	"context"
	"testing"

	"github.com/drone/drone-go/drone"
	"github.com/drone/drone-go/plugin/secret"

	"github.com/ericchiang/k8s"

	"github.com/google/go-cmp/cmp"
	"github.com/h2non/gock"
)

var noContext = context.Background()

func TestPlugin(t *testing.T) {
	defer gock.Off()

	client := &k8s.Client{
		Endpoint:  "http://localhost",
		Namespace: "default",
	}

	gock.New("http://localhost").
		Reply(200).
		AddHeader("Content-Type", "application/vnd.kubernetes.protobuf").
		File("testdata/secret.protobuf")

	req := &secret.Request{
		Name: "username",
		Path: "docker",
		Build: drone.Build{
			Event: "push",
		},
		Repo: drone.Repo{
			Slug: "octocat/hello-world",
		},
	}
	plugin := New(client, client.Namespace)
	got, err := plugin.Find(noContext, req)
	if err != nil {
		t.Error(err)
		return
	}

	want := &drone.Secret{
		Name: "username",
		Data: "admin",
		Pull: true,
		Fork: true,
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf(diff)
		return
	}

	if gock.IsPending() {
		t.Errorf("Unfinished requests")
		return
	}
}

func TestPlugin_FilterRepo(t *testing.T) {
	defer gock.Off()

	client := &k8s.Client{
		Endpoint:  "http://localhost",
		Namespace: "default",
	}

	gock.New("http://localhost").
		Reply(200).
		AddHeader("Content-Type", "application/vnd.kubernetes.protobuf").
		File("testdata/secret.protobuf")

	req := &secret.Request{
		Name: "username",
		Path: "docker",
		Build: drone.Build{
			Event: "push",
		},
		Repo: drone.Repo{
			Slug: "spaceghost/hello-world",
		},
	}
	plugin := New(client, client.Namespace)
	_, err := plugin.Find(noContext, req)
	if err == nil {
		t.Errorf("Expect error")
		return
	}
	if want, got := err.Error(), "access denied: repository does not match"; got != want {
		t.Errorf("Want error %q, got %q", want, got)
		return
	}

	if gock.IsPending() {
		t.Errorf("Unfinished requests")
		return
	}
}

func TestPlugin_FilterEvent(t *testing.T) {
	defer gock.Off()

	client := &k8s.Client{
		Endpoint:  "http://localhost",
		Namespace: "default",
	}

	gock.New("http://localhost").
		Reply(200).
		AddHeader("Content-Type", "application/vnd.kubernetes.protobuf").
		File("testdata/secret.protobuf")

	req := &secret.Request{
		Name: "username",
		Path: "docker",
		Build: drone.Build{
			Event: "pull_request",
		},
		Repo: drone.Repo{
			Slug: "octocat/hello-world",
		},
	}
	plugin := New(client, client.Namespace)
	_, err := plugin.Find(noContext, req)
	if err == nil {
		t.Errorf("Expect error")
		return
	}
	if want, got := err.Error(), "access denied: event does not match"; got != want {
		t.Errorf("Want error %q, got %q", want, got)
		return
	}

	if gock.IsPending() {
		t.Errorf("Unfinished requests")
		return
	}
}

func TestPlugin_MissingPath(t *testing.T) {
	req := &secret.Request{
		Name: "password",
	}
	_, err := New(nil, "default").Find(noContext, req)
	if err == nil {
		t.Errorf("Expect invalid path error")
		return
	}
	if got, want := err.Error(), "invalid or missing secret path"; got != want {
		t.Errorf("Want error message %s, got %s", want, got)
	}
}

func TestPlugin_MissingName(t *testing.T) {
	req := &secret.Request{
		Path: "docker",
	}
	_, err := New(nil, "default").Find(noContext, req)
	if err == nil {
		t.Errorf("Expect invalid path error")
		return
	}
	if got, want := err.Error(), "invalid or missing secret name"; got != want {
		t.Errorf("Want error message %s, got %s", want, got)
	}
}

func TestPlugin_NotFound(t *testing.T) {
	defer gock.Off()

	client := &k8s.Client{
		Endpoint:  "http://localhost",
		Namespace: "default",
	}

	gock.New("http://localhost").
		Reply(404).
		AddHeader("Content-Type", "application/vnd.kubernetes.protobuf").
		File("testdata/error.protobuf")

	req := &secret.Request{
		Name: "username",
		Path: "docker",
		Build: drone.Build{
			Event: "push",
		},
		Repo: drone.Repo{
			Slug: "octocat/hello-world",
		},
	}
	plugin := New(client, client.Namespace)
	_, err := plugin.Find(noContext, req)
	if _, ok := err.(*k8s.APIError); !ok {
		t.Errorf("Expect APIError")
		return
	}

	if gock.IsPending() {
		t.Errorf("Unfinished requests")
		return
	}
}

func TestPlugin_InvalidAttribute(t *testing.T) {
	defer gock.Off()

	client := &k8s.Client{
		Endpoint:  "http://localhost",
		Namespace: "default",
	}

	gock.New("http://localhost").
		Reply(200).
		AddHeader("Content-Type", "application/vnd.kubernetes.protobuf").
		File("testdata/secret.protobuf")

	req := &secret.Request{
		Name: "token",
		Path: "docker",
		Build: drone.Build{
			Event: "push",
		},
		Repo: drone.Repo{
			Slug: "octocat/hello-world",
		},
	}
	plugin := New(client, client.Namespace)
	_, err := plugin.Find(noContext, req)
	if err == nil {
		t.Errorf("Expect secret not found error")
		return
	}
	if got, want := err.Error(), "secret not found"; got != want {
		t.Errorf("Want error message %s, got %s", want, got)
	}
}
