h2c - HTTP/2 client
-----------------------

`h2c` is a simple command line client for HTTP/2 servers.

It is currently in a very early stage. The best way to learn about it is to read the blog posts on [http://blog.http2client.net](http://blog.http2client.net).

The basic usage is as follows:

```bash
h2c start &
h2c connect 127.0.0.1:443
h2c get /index.html
h2c stop
```

How to Download and Run
-----------------------

Binary releases are available on the [GitHub Releases](https://github.com/fstab/h2c/releases).

1. Download the latest release ZIP file: [h2c-v0.0.4.zip](https://github.com/fstab/h2c/releases/download/v0.0.4/h2c-v0.0.4.zip)
2. Extract the ZIP file
3. Find the executable for your system in the `bin` folder:
  * Linux: `h2c_linux_amd64`
  * OS X: `h2c_darwin_amd64`
  * Windows: `h2c_windows_amd64.exe`
4. Rename that executable to `h2c`, or `h2c.exe` on Windows
5. Move the executable into a folder on your PATH.

How to Build from Source
------------------------

`h2c` is developed with [go 1.4.2](https://golang.org/dl/). With [go](https://golang.org) set up, you can download, compile, and install `h2c` as follows:

```bash
go get github.com/fstab/h2c
```

Related Work
------------

`h2c` is built using Brad Fitzpatrick's [HTTP/2 support for Go](https://github.com/bradfitz/http2). There is an HTTP/2 console debugger included in [bradfitz/http2](https://github.com/bradfitz/http2), but just like `h2c`, it is currently only a quick few hour hack, so it is hard to tell if they aim at the same kind of tool.

LICENSE
-------

`h2c` is licensed under the [Apache License, Version 2.0](LICENSE).

`h2c` is built using [Go](https://golang.org/) and [bradfitz/http2](https://github.com/bradfitz/http2). Both are licensed under [Google's Go license](https://code.google.com/p/go/source/browse/LICENSE).
