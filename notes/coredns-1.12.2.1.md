+++
title = "CoreDNS-1.12.2.1 Release"
description = "CoreDNS-1.12.2.1 Release Notes."
tags = ["Release", "1.12.2.1", "Notes"]
release = "1.12.2.1"
date = "2024-06-28T00:00:00+00:00"
author = "coredns"
+++

This release updates the plugin.cfg file to change the source of the https plugin.

## HTTPS plugin

### HTTP/3 Support

Added support for HTTP/3 in the HTTPS plugin by introducing a new httpVersion configuration property in setup.go.
Implemented logic to configure HTTP clients based on the specified httpVersion (defaulting to HTTP/2) and added QUIC-specific settings for HTTP/3.

### Configuration Enhancements

Extended the httpsConfig struct to include an httpVersion field, defaulting to HTTP2.0.
Added a new http_version directive to the CoreDNS configuration parser, allowing users to specify HTTP/2 or HTTP/3.