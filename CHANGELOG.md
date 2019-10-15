# Changelog

This file documents all notables changes to the project.
The project uses [semantic versioning](https://semver.org/spec/v2.0.0.html) and is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

## [0.0.1]

### Added

- Initial release after code audit
- Added documentation, README, Code of Conduct, Contributing Guidelines, and Changelog
- Basic features of the crawler are implemented :
  - package is a module, but contains a compilable app in app/crawl.go
  - public functions are FetchLinks(), StreamLinks() and ScrapLinks()
  - documentation on https://godoc.org/github.com/bytemare/crawl
  - single domain scope
  - parallel scraping for speed, without critical code in concurrent goroutines
  - optional timeout
  - scraps queries and fragments from URLs
  - control plane for signal interception and timeout 
  - avoid loops on already visited links and visiting links
  - logging through logrus, and logs to file in JSON for log aggregation
- added some code examples in README
- integrated CI tools

[Unreleased]: https://github.com/olivierlacan/keep-a-changelog/compare/v0.0.1...HEAD
[0.0.1]: https://github.com/bytemare/crawl/releases/tag/v0.1.0