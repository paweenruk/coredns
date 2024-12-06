+++
title = "CoreDNS-1.12.0.1 Release"
description = "CoreDNS-1.12.0.1 Release Notes."
tags = ["Release", "1.12.0.1", "Notes"]
release = "1.12.0.1"
date = "2024-12-06T00:00:00+00:00"
author = "coredns"
+++

This release includes several updates to the Docker workflow configuration, versioning, and plugin configuration for the CoreDNS project. The most important changes include modifying the Docker build and publish commands, updating the CoreDNS version, and adding new plugins.

## Docker workflow updates:
* [`.github/workflows/docker.yml`](diffhunk://#diff-3f5366f6d6df3ec1179e5efadc6f350bfa88eebf4e2da589b4d94ccb85ae5e94L25-R25): Updated the Docker build and publish commands to use `paweenruk` as the Docker repository and added a GitHub release URL. [[1]](diffhunk://#diff-3f5366f6d6df3ec1179e5efadc6f350bfa88eebf4e2da589b4d94ccb85ae5e94L25-R25) [[2]](diffhunk://#diff-3f5366f6d6df3ec1179e5efadc6f350bfa88eebf4e2da589b4d94ccb85ae5e94L34-R34)

## Version update:
* [`coremain/version.go`](diffhunk://#diff-179a1bdcf9fe4d4675e4cfe57406cb70b75de872afc69a2ae308ae4619a2741aL5-R5): Updated the CoreDNS version from `1.12.0` to `1.12.0.1`.

## Plugin configuration:
* [`plugin.cfg`](diffhunk://#diff-b4e59ee676115519545c5bedf117fdff14506f9429fa10214be69a1352a87e0fR75-R76): Added `fanout` and `https` plugins to the configuration.