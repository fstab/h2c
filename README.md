h2c - HTTP/2 client
-----------------------

`h2c` is a simple command line client for HTTP/2 servers.

It is currently in a very early stage. The best way to learn about it is to read the blog posts on [http://fstab.github.io/h2c/](http://fstab.github.io/h2c/).

The basic usage is as follows:

```bash
h2c start &
h2c connect 127.0.0.1:443
h2c get /index.html
h2c stop
```

How to Download and Run
-----------------------

TODO

How to Build from Source
------------------------

`h2c` is developed with [go 1.4.2](https://golang.org/dl/). With [go](https://golang.org) set up, you can download, compile, and install `h2c` as follows:

```bash
go install github.com/fstab/h2c
```

Related Work
------------

`h2c` is built using Brad Fitzpatrick's excellent [http2 support for Go](https://github.com/bradfitz/http2). There is an HTTP/2 console debugger included in [bradfitz/http2](https://github.com/bradfitz/http2), but just like `h2c`, it is currently only a quick few hour hack, so it is hard to tell if they aim at the same kind of tool.

LICENSE
-------

`h2c` is licensed under the [Apache License, Version 2.0](LICENSE).

`h2c` is built using [Go](https://golang.org/) and [bradfitz/http2](https://github.com/bradfitz/http2). Both are licensed under [Google's Go license](https://code.google.com/p/go/source/browse/LICENSE).

Who We Are
----------

TODO
