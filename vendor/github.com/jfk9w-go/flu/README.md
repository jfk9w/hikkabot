## flu

[![ci](https://github.com/jfk9w-go/flu/actions/workflows/ci.yml/badge.svg?branch=master)](https://github.com/jfk9w-go/flu/actions/workflows/ci.yml) 
[![Go Reference](https://pkg.go.dev/badge/github.com/jfk9w-go/flu@master.svg)](https://pkg.go.dev/github.com/jfk9w-go/flu)

This package contains everything I need to build Go applications quickly at least for personal use.
It provides some common functionality like:
* safer IO operations & abstractions wrappers, JSON/YAML type wrappers
* fluent HTTP request making (**httpf**)
* dependency injection and application configuration framework (**apfel**)
* metrics reporting for Prometheus and Graphite (**me3x**)
* synchronization primitives and utilities (**syncf**)
* [gorm.io/gorm](https://github.com/go-gorm/gorm) extensions & utilities (**gormf**)
* logging framework based on **log** (**logf**)
* "collection" utilities (**colf**)
* retry with backoff implementation (**backoff**)
* ...and probably more

Please see [godoc](https://pkg.go.dev/github.com/jfk9w-go/flu) for more information.

### Test coverage

It *is* pretty low, but enough for use in pet projects.
I will try to extend test coverage, but it may take a while until the package 
may be considered "production-ready" (although I do use the vast majority,
if not all, of the provided functionality in my pet projects, namely, [homebot](https://github.com/jfk9w-go/homebot)
and [hikkabot](https://github.com/jfk9w/hikkabot), and some libraries like 
[telegram-bot-api](https://github.com/jfk9w-go/telegram-bot-api) and [aconvert-api](https://github.com/jfk9w-go/aconvert-api)).

### API stability

This package is (sometimes) under active development and API may change without
notice. Although this should happen only if some fatal flaws are found.
