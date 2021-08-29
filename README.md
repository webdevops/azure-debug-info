Azure debug info
==============================

[![license](https://img.shields.io/github/license/webdevops/azure-debug-info.svg)](https://github.com/webdevops/azure-debug-info/blob/master/LICENSE)
[![DockerHub](https://img.shields.io/badge/DockerHub-webdevops%2Fazure--debug--info-blue)](https://hub.docker.com/r/webdevops/azure-debug-info/)
[![Quay.io](https://img.shields.io/badge/Quay.io-webdevops%2Fazure--debug--info-blue)](https://quay.io/repository/webdevops/azure-debug-info)

Application to get details about assigned ServicePrincipal (eg. MSI/aad-pod-identity) and
lists which Azure Subscriptions and Azure ResourceGroups it can find.

## Features

- logs ServicePrincipal information
- MSI/aad-pod-identity support
- logs all visible Azure Subscriptions
- logs all visible Azure ResourceGroups

Configuration
-------------

```
Usage:
  azure-debug-info [OPTIONS]

Application Options:
      --debug              debug mode [$DEBUG]
  -v, --verbose            verbose mode [$VERBOSE]
      --log.json           Switch log output to json format [$LOG_JSON]
      --azure-environment= Azure environment name (default: AZUREPUBLICCLOUD) [$AZURE_ENVIRONMENT]

Help Options:
  -h, --help               Show this help message
```

for Azure API authentication (using ENV vars) see https://github.com/Azure/azure-sdk-for-go#authentication
