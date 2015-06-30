---
layout:   post
title:    "Analyzing HTTP/2 Traffic"
date:     2015-07-01 17:15:21
comments: true
---

HTTP/2 is a binary protocol. Tool support is needed to analyze HTTP/2 traffic.
This post introduces `h2c start --dump`, an option for [h2c] that intercepts
HTTP/2 traffic and prints it to the console.

TL;DR
-----

This post explains the output of the `h2c start --dump` command.

Hello, World! Again.
--------------------

As shown in the [Hello, World!] post, the [h2c] tool relies on a _h2c process_
running in the background. This process is started with the `h2c start` command
and, among other things, maintains the connection to the HTTP server.

The option `h2c start --dump` prints all HTTP/2 traffic to the console.

![h2c start --dump]( {{site.url}}{{site.baseurl}}/assets/2015-07-01-h2c-dump.png)

The screenshot above shows the dump of the [Hello, World!] example from the first
blog post:

{% highlight bash %}
h2c connect 192.168.59.103:8443
h2c get /hello-world/api/hello-world
{% endhighlight %}

Analyzing the Output
-------------------

### <font color="#505050">-&gt;</font> / <font color="#505050">&lt;-</font>

The arrows show if the frame is incoming `<-` or outgoing `->`.

### <font color="#008B8B">SETTINGS(0)</font> / <font color="#008B8B">HEADERS(1)</font> / <font color="#008B8B">DATA(1)</font>

The headlines show the frame type, and the stream ID. Each [stream] corresponds to a
request/response pair in HTTP/1. The example above shows two streams:

  * An initial stream with ID 0 for negotiating the settings when the connection
    is established. This has no equivalent in HTTP/1.
  * A stream with ID 1 for the GET request and response. The request is a single
    outgoing `->` [HEADERS] frame, the response is composed of an incoming `<-` [HEADERS]
    frame and an incoming `<-` [DATA] frame.

### <font color="#006400">+ ACK</font> / <font color="#006400">+ END_STREAM</font> / <font color="#006400">+ END_HEADERS</font>

Some frame types support _flags_ that can be either set `+` or clear `-`.
The END_HEADERS flag for the [HEADERS] frame defines if all headers are included in
the frame, or if the headers are continued with a [CONTINUATION] frame.
As the headers in the example fit into single HEADERS frames, the END_HEADERS flag is
always set `+`.

The END_STREAM flag indicates if there will be more frames for the stream. The
request in the example consists only of a HEADERS frame, so END_STREAM is set `+`
for the outgoing `->` [HEADERS] frame. The response consists of a [HEADERS] frame and a
[DATA] frame, so the END_STREAM flag is clear `-` for the incoming `<-` [HEADERS] frame,
and it is set `+` for the incoming `<-` [DATA] frame.

### <font color="#0000AA">name:</font> <font color="#505050">value</font>

The actual content of the frame is shown as name/value pairs. For [SETTINGS] frames,
these are the settings. For [HEADERS] frames, these are the headers.

Current Status
--------------

[h2c] is currently in a very early state. As of [release v0.0.5], the only frames
implemented are the frames shown above, and the only interaction implemented is the
GET request.

The focus in the next few days will be on implementing POST and PUT requests.
The next feature after that should be support for [server push].

[h2c]: https://github.com/fstab/h2c
[Hello, World!]: http://blog.http2client.net/2015/06/07/http2-hello-world.html
[stream]: https://httpwg.github.io/specs/rfc7540.html#StreamsLayer
[HEADERS]: https://httpwg.github.io/specs/rfc7540.html#HEADERS
[CONTINUATION]: https://httpwg.github.io/specs/rfc7540.html#CONTINUATION
[DATA]: https://httpwg.github.io/specs/rfc7540.html#DATA
[SETTINGS]: https://httpwg.github.io/specs/rfc7540.html#SETTINGS
[release v0.0.5]: https://github.com/fstab/h2c/releases
[server push]: https://httpwg.github.io/specs/rfc7540.html#PushResources
