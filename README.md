# jx-role-controller

[![Go Report Card](https://goreportcard.com/badge/github.com/jenkins-x/jx-role-controller)](https://goreportcard.com/report/github.com/jenkins-x/jx-role-controller)

Used to deploy a Jenkins X Role controller within a kubernetes cluster. The role controller is responsible for watching roles/rolebindings/environmentrolebindings and applying these across environments/namespaces.

Can be added within a Jenkins X dev environment by making the following changes:

1. Add the chart into the environment requirements file.

```yaml
- name: jx-role-controller
  repository: https://storage.googleapis.com/chartmuseum.jenkins-x.io
  version: 0.0.19
```

2. Add the associating directory for any chart overrides within the `env` directory..

```sh
env
├── Chart.yaml
├── Makefile
├── ...
├── jx-role-controller
│   └── values.tmpl.yaml
├── ...
├── requirements.yaml
└── values.tmpl.yaml
```

Part of Jenkins X shared components.

For more information on configuring logging file, formats and levels see the [Jenkins X logging](https://github.com/jenkins-x/jx-logging) component.
