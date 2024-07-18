[![Sensu Bonsai Asset](https://img.shields.io/badge/Bonsai-Download%20Me-brightgreen.svg?colorB=89C967&logo=sensu)](https://bonsai.sensu.io/assets/yourusername/sensu-check-http-json)
![goreleaser](https://github.com/yourusername/sensu-check-http-json/workflows/goreleaser/badge.svg)

# sensu-check-http-json

## Table of Contents

- [Overview](#overview)
- [Files](#files)
- [Usage examples](#usage-examples)
- [Configuration](#configuration)
  - [Asset registration](#asset-registration)
  - [Check definition](#check-definition)
- [Installation from source](#installation-from-source)

## Overview

This Sensu plugin performs HTTP checks and extracts values from JSON responses using jq queries. It supports basic comparison operations and handles TLS verification options.

## Files

- sensu-check-http-json

## Usage examples

```bash
$ sensu-check-http-json -u https://jsonplaceholder.typicode.com/posts/1 -q "userId" -e "> 1"

OK: URL https://jsonplaceholder.typicode.com/posts/1, extracted value: 1, expression check passed: 1 > 1
```

Help:

```bash
$ sensu-check-http-json -h

A Sensu plugin to perform HTTP checks and extract values from JSON responses.

Usage:
  sensu-check-http-json [flags]
  sensu-check-http-json [command]

Available Commands:
  help        Help about any command
  version     Print the version number of this plugin

Flags:
  -d, --debug                  Enable debug mode
  -e, --expression string      Expression for comparing result of query
  -h, --help                   help for sensu-check-http-json
  -i, --insecure-skip-verify   Skip TLS certificate verification (not recommended!)
  -q, --query string           Query for extracting value from JSON
  -T, --timeout int            Request timeout in seconds (default 15)
  -u, --url string             URL to test (default "http://localhost:80/")
```

## Configuration

### Asset registration

[Sensu Assets][10] are the best way to make use of this plugin. If you're not using an asset, please consider doing so! If you're using sensuctl 5.13 with Sensu Backend 5.13 or later, you can use the following command to add the asset:

```bash
sensuctl asset add DoctorOgg/sensu-check-http-json
```

If you're using an earlier version of sensuctl, you can find the asset on the [Bonsai Asset Index][https://bonsai.sensu.io/assets/yourusername/sensu-check-http-json].

### Check definition

```yml
---

type: CheckConfig
api_version: core/v2
metadata:
  name: sensu-check-http-json
  namespace: default
spec:
  command: sensu-check-http-json -u <https://jsonplaceholder.typicode.com/posts/1> -q "userId" -e "> 1"
  subscriptions:

- system
  runtime_assets:
- DoctorOgg/sensu-check-http-json
```
