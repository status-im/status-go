🚌 message-bus
================
[![Build Status](https://travis-ci.org/vardius/message-bus.svg?branch=master)](https://travis-ci.org/vardius/message-bus)
[![Go Report Card](https://goreportcard.com/badge/github.com/vardius/message-bus)](https://goreportcard.com/report/github.com/vardius/message-bus)
[![codecov](https://codecov.io/gh/vardius/message-bus/branch/master/graph/badge.svg)](https://codecov.io/gh/vardius/message-bus)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fvardius%2Fmessage-bus.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fvardius%2Fmessage-bus?ref=badge_shield)
[![](https://godoc.org/github.com/vardius/message-bus?status.svg)](https://pkg.go.dev/github.com/vardius/message-bus)
[![license](https://img.shields.io/github/license/mashape/apistatus.svg)](https://github.com/vardius/message-bus/blob/master/LICENSE.md)

<img align="right" height="180px" src="website/src/static/img/logo.png" alt="logo" />

Go simple async message bus.

📖 ABOUT
==================================================
Contributors:

* [Rafał Lorenz](http://rafallorenz.com)

Want to contribute ? Feel free to send pull requests!

Have problems, bugs, feature ideas?
We are using the github [issue tracker](https://github.com/vardius/message-bus/issues) to manage them.

## 📚 Documentation

For **documentation** (_including examples_), **visit [rafallorenz.com/message-bus](http://rafallorenz.com/message-bus)**

For **GoDoc** reference, **visit [pkg.go.dev](https://pkg.go.dev/github.com/vardius/message-bus)**

🚏 HOW TO USE
==================================================

## 🚅 Benchmark

```bash
➜  message-bus git:(master) ✗ go test -bench=. -cpu=4 -benchmem
goos: darwin
goarch: amd64
pkg: github.com/vardius/message-bus
BenchmarkPublish-4                   	 4430224	       250 ns/op	       0 B/op	       0 allocs/op
BenchmarkSubscribe-4                 	  598240	      2037 ns/op	     735 B/op	       5 allocs/op
```

👉 **[Click here](https://rafallorenz.com/message-bus/docs/benchmark)** to see all benchmark results.

## Features
- [Documentation](https://rafallorenz.com/message-bus/)

🚏 HOW TO USE
==================================================

- [Basic example](https://rafallorenz.com/message-bus/docs/basic-example)
- [Pub/Sub](https://rafallorenz.com/message-bus/docs/pubsub)

📜 [License](LICENSE.md)
-------

This package is released under the MIT license. See the complete license in the package:

[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fvardius%2Fmessage-bus.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fvardius%2Fmessage-bus?ref=badge_large)
