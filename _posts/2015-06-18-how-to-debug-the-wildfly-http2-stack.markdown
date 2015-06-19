---
layout:   post
title:    "How to Debug Wildfly's HTTP/2 Stack"
date:     2015-06-18 17:24:58
comments: true
---

Debugging the inner workings of an application server is something Java developers usually don't do.

In the [Hello, World!] blog post we used a [Docker] image with [Wildfly 9.0.0.Beta1 with HTTP/2](https://registry.hub.docker.com/u/fstab/wildfly-http2) support pre-installed.
This is great for deploying and debugging a WAR file, but it is not easy to debug into implementation of the container itself to learn how [HTTP/2] is handled on a lower level.

TL;DR
-----

In this post we will

  * Run Wildfly's Web server [Undertow] as a stand-alone plain Java program with a `main()` method.
  * Have a Hello, World! Servlet deployed.
  * Be able to set break points and debug into the internal [HTTP/2] protocol implementation

Undertow
--------

[Wildfly] is composed of subsystems. The subsystem providing Wildfly's Web server is called [Undertow], and it can be run as a stand-alone Web server independently of Wildfly.
In order to look into the [HTTP/2] protocol implementation, we start a stand-alone Undertow instance without any other Wildfly subsystems.

I created an [undertow-http2-servlet-example] which can be downloaded and run as follows:

{% highlight bash %}
git clone https://github.com/fstab/http2-examples
cd http2-examples/undertow-http2-servlet-example
mvn clean package
java -jar target/undertow-http2-servlet-example.jar
{% endhighlight %}

Just like in the [Hello, World!] post, the example service can be accessed on [https://localhost:8443/hello-world/api/hello-world].

![Example Servlet in Google Chrome]({{site.url}}{{site.baseurl}}/assets/2015-06-18-undertow-http2-servlet-example-in-browser.png)

In an [IDE], the example project's main class `de.consol.labs.h2c.Http2Server` can also be run using the
<img src="{{site.url}}{{site.baseurl}}/assets/2015-06-18-idea-play-button.png" alt="play" style="height: 26px;"/> button, without using [maven].

Breakpoints
-----------

The [HTTP/2] protocol is implemented in `undertow-core-1.2.7.Final.jar`, which is included with the dependencies in the example project's `pom.xml` file.
Any Java IDE should be able to open the Java sources for this dependency and set breakpoints there.
The parser for [HTTP/2 frames] can be found in package `io.undertow.protocols.http2`, the server code handling HTTP/2 communication is in `io.undertow.server.protocol.http2`.

What's Next?
------------

Using stand-alone [Undertow] with a `main()` method makes it easy to debug the
inner workings of the Web server's [HTTP/2] implementation.

In the next posts, we will learn how [HTTP/2] streams are handled and mapped to Servlet calls.

[Hello, World!]: /2015/06/07/http2-hello-world.html
[Docker]: https://www.docker.com
[Wildfly]: http://wildfly.org
[undertow-http2-servlet-example]: https://github.com/fstab/http2-examples/tree/master/undertow-http2-servlet-example
[undertow]: http://undertow.io
[IDE]: https://www.jetbrains.com/idea
[maven]: https://maven.apache.org
[https://localhost:8443/hello-world/api/hello-world]: https://localhost:8443/hello-world/api/hello-world
[HTTP/2]: https://http2.github.io
[HTTP/2 frames]: https://httpwg.github.io/specs/rfc7540.html#FrameTypes
