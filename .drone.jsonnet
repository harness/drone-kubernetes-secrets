# generates the .drone.yml file
#   jsonnet -S .drone.jsonnet > .drone.jsonnet.yml

# docker secrets requested for the pipeline. The can be
# used with the docker and manifest plugins to authenticate
# to the docker.io registry.
local secrets = {
  docker_username: {
    external: {
      name: 'drone/docker#username',
    },
  },
  docker_password: {
    external: {
      name: 'drone/docker#password',
    },
  },
};

# pipeline creates a new build pipeline based on the
# operating system and architecture.
local pipeline(os='', arch='', variant=null, kernel=null) = {
  workspace: {
    base: '/go',
    path: 'src/github.com/drone/drone-kubernetes-secrets',
  },
  metadata: {
      name: std.join('-', [os, arch]),
  },
  platform: {
      os: os,
      arch: arch,
      variant: variant,
      kernel: kernel,
  },
  pipeline: [
    {
      build: {
        image: 'golang:1.10',
        commands: [
          'go get -u github.com/golang/dep/cmd/dep',
          'dep ensure',
          'dep status',
          'go test -v -cover ./...',
          'CGO_ENABLED=0 go build -o release/' + os + '/' + arch + '/drone-kubernetes-secrets ./cmd/drone-kubernetes-secrets',
        ],
      }, 
    },
    # build and publish the docker image to the registry.
    # The image is tagged with the os and architecture.
    {
      publish: {
        image: 'plugins/docker:17.12',
        secrets: [
          field for field
            in std.objectFields(secrets)
        ],
        repo: 'drone/drone-kubernetes',
        dockerfile: std.join('.', ['docker/Dockerfile', os, arch]),
        auto_tag: true,
        auto_tag_prefix: std.join('-', [os, arch]),
      }, 
    },
  ],
  secrets: secrets,
};

# generate parallel pipeline steps that execute for each
# supported os and architecture.
local pipelines = [
  pipeline(os='linux', arch='amd64'),
  pipeline(os='linux', arch='arm', variant=7),
  pipeline(os='linux', arch='arm64', variant=8),
];

std.manifestYamlStream(pipelines + 
  [{
    metadata: {
        name: 'manifest',
    },
    pipeline: [
      {
        manifest: {
          image: 'plugins/manifest:1',
          spec: 'manifest.tmpl',
          auto_tag: true,
          ignore_missing: true,
          secrets: [
            field for field
              in std.objectFields(secrets)
          ],
        },
      }
    ],
    secrets: secrets,
    # the manifest pipeline should only be triggered
    # for push and tag events.
    trigger: {
      event: [
        'push',
        'tag',
      ],
    },
    # the manifest pipeline should only execute after
    # all build pipelines are complete and the platform-
    # specific docker images are published.
    depends_on: [
      x.metadata.name for x in pipelines
    ],
  }]
)
