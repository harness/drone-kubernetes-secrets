local windows_pipe = '\\\\\\\\.\\\\pipe\\\\docker_engine';
local windows_pipe_volume = 'docker_pipe';
local test_pipeline_name = 'testing';

local windows(os) = os == 'windows';

local golang_image(os, version) =
  'golang:' + '1.13' + if windows(os) then '-windowsservercore-' + version else '';

{
  test(os='linux', arch='amd64', version='')::
    local is_windows = windows(os);
    local golang = golang_image(os, version);
    local volumes = if is_windows then [{name: 'gopath', path: 'C:\\\\gopath'}] else [{name: 'gopath', path: '/go',}];
    {
      kind: 'pipeline',
      name: test_pipeline_name,
      platform: {
        os: os,
        arch: arch,
        version: if std.length(version) > 0 then version,
      },
      steps: [
        {
          name: 'vet',
          image: golang,
          pull: 'always',
          environment: {
            GO111MODULE: 'on',
          },
          commands: [
            'go vet ./...',
          ],
          volumes: volumes,
        },
        {
          name: 'test',
          image: golang,
          pull: 'always',
          environment: {
            GO111MODULE: 'on',
          },
          commands: [
            'go test -cover ./...',
          ],
          volumes: volumes,
        },
      ],
      trigger: {
        ref: [
          'refs/heads/master',
          'refs/tags/**',
          'refs/pull/**',
        ],
      },
      volumes: [{name: 'gopath', temp: {}}]
    },

  build(name, os='linux', arch='amd64', version='')::
    local is_windows = windows(os);
    local tag = if is_windows then os + '-' + version else os + '-' + arch;
    local file_suffix = std.strReplace(tag, '-', '.');
    local volumes = if is_windows then [{ name: windows_pipe_volume, path: windows_pipe }] else [];
    local golang = golang_image(os, version);
    local docker_name = 'drone/' + std.splitLimit(name, '-', 1)[1];
    local extension = if is_windows then '.exe' else '';
    {
      kind: 'pipeline',
      name: tag,
      platform: {
        os: os,
        arch: arch,
        version: if std.length(version) > 0 then version,
      },
      steps: [
        {
          name: 'build',
          image: golang,
          pull: 'always',
          environment: {
            CGO_ENABLED: '0',
            GO111MODULE: 'on',
          },
          commands: [
            'go build -v -a -tags netgo -o release/' + os + '/' + arch + '/' + name + extension + ' ./cmd/' + name,
          ],
        },
        {
          name: 'dryrun',
          image: 'plugins/docker:' + tag,
          pull: 'always',
          settings: {
            dry_run: true,
            tags: tag,
            dockerfile: 'docker/Dockerfile.' + file_suffix,
            daemon_off: if is_windows then 'true' else 'false',
            repo: docker_name,
            username: { from_secret: 'docker_username' },
            password: { from_secret: 'docker_password' },
          },
          volumes: if std.length(volumes) > 0 then volumes,
          when: {
            event: ['pull_request'],
          },
        },
        {
          name: 'publish',
          image: 'plugins/docker:' + tag,
          pull: 'always',
          settings: {
            auto_tag: true,
            auto_tag_suffix: tag,
            daemon_off: if is_windows then 'true' else 'false',
            dockerfile: 'docker/Dockerfile.' + file_suffix,
            repo: docker_name,
            username: { from_secret: 'docker_username' },
            password: { from_secret: 'docker_password' },
          },
          volumes: if std.length(volumes) > 0 then volumes,
          when: {
            event: {
              exclude: ['pull_request'],
            },
          },
        },
        {
          name: 'tarball',
          image: golang,
          pull: 'always',
          commands: [
            'tar -cvzf release/' + name + '_' + os + '_' + arch + '.tar.gz -C release/' + os + '/' + arch + ' ' + name,
            'sha256sum release/' + name + '_' + os + '_' + arch + '.tar.gz > release/' + name + '_' + os + '_' + arch + '.tar.gz.sha256'
          ],
          when: {
            event: ['tag'],
          },
        },
        {
          name: 'gpgsign',
          image: 'plugins/gpgsign',
          pull: 'always',
          settings: {
            files: [
              'release/*.tar.gz',
              'release/*.tar.gz.sha256',
            ],
            key: { from_secret: 'gpgsign_key' },
            passphrase: { from_secret: 'gpgkey_passphrase' },
          },
          when: {
            event: ['tag'],
          },
        },
        {
          name: 'github',
          image: 'plugins/github-release',
          pull: 'always',
          settings: {
            files: [
              'release/*.tar.gz',
              'release/*.tar.gz.sha256',
              'release/*.tar.gz.asc',
            ],
            token: { from_secret: 'github_token' },
          },
          when: {
            event: ['tag'],
          },
        },
      ],
      trigger: {
        ref: [
          'refs/heads/master',
          'refs/tags/**',
          'refs/pull/**',
        ],
      },
      depends_on: [test_pipeline_name],
      volumes: if is_windows then [{ name: windows_pipe_volume, host: { path: windows_pipe } }],
    },

  notifications(os='linux', arch='amd64', version='', depends_on=[])::
    {
      kind: 'pipeline',
      name: 'notifications',
      platform: {
        os: os,
        arch: arch,
        version: if std.length(version) > 0 then version,
      },
      steps: [
        {
          name: 'manifest',
          image: 'plugins/manifest',
          pull: 'always',
          settings: {
            username: { from_secret: 'docker_username' },
            password: { from_secret: 'docker_password' },
            spec: 'docker/manifest.tmpl',
            ignore_missing: true,
          },
        },
      ],
      trigger: {
        ref: [
          'refs/heads/master',
          'refs/tags/**',
        ],
      },
      depends_on: depends_on,
    },
}
