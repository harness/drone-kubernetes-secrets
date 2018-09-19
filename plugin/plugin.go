// Copyright 2018 Drone.IO Inc
// Use of this software is governed by the Business Source License
// that can be found in the LICENSE file.

package plugin

import (
	"context"
	"errors"
	"fmt"

	"github.com/drone/drone-go/drone"
	"github.com/drone/drone-go/plugin/secret"

	"github.com/ericchiang/k8s"
	"github.com/ericchiang/k8s/apis/core/v1"
)

// New returns a new secret plugin that sources secrets
// from the Kubernetes secrets manager.
func New(client *k8s.Client, namespace string) secret.Plugin {
	return &plugin{
		namespace: namespace,
		client:    client,
	}
}

type plugin struct {
	client    *k8s.Client
	namespace string
}

func (p *plugin) Find(ctx context.Context, req *secret.Request) (*drone.Secret, error) {
	fmt.Printf("%+v\n", req)

	if req.Path == "" {
		return nil, errors.New("invalid or missing secret path")
	}
	if req.Name == "" {
		return nil, errors.New("invalid or missing secret name")
	}

	path := req.Path
	name := req.Name

	// makes an api call to the kubernetes secrets manager and
	// attempts to retrieve the secret at the requested path.
	var secret v1.Secret
	err := p.client.Get(ctx, p.namespace, path, &secret)
	if err != nil {
		return nil, err
	}
	data, ok := secret.Data[name]
	if !ok {
		return nil, errors.New("secret not found")
	}

	// the user can filter out requets based on event type
	// using the X-Drone-Events secret key. Check for this
	// user-defined filter logic.
	events := extractEvents(secret.Metadata.Annotations)
	if !match(req.Build.Event, events) {
		return nil, errors.New("access denied: event does not match")
	}

	// the user can filter out requets based on repository
	// using the X-Drone-Repos secret key. Check for this
	// user-defined filter logic.
	repos := extractRepos(secret.Metadata.Annotations)
	if !match(req.Repo.Slug, repos) {
		return nil, errors.New("access denied: repository does not match")
	}

	return &drone.Secret{
		Name: name,
		Data: string(data),
		Pull: true, // always true. use X-Drone-Events to prevent pull requests.
		Fork: true, // always true. use X-Drone-Events to prevent pull requests.
	}, nil
}
