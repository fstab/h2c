---
layout: post
title:  "HTTP/2 Hello, World!"
date:   2015-06-07 13:40:54
---

[HTTP/2], the new version of the HTTP protocol, provides a lot of interesting features for REST-based server-to-server communication:

  * bidirectional communication using push requests
  * multiplexing within a single TCP connection
  * long running connections
  * stateful connections
  * etc.

HTTP/2 does not define a JavaScript API, so JavaScript clients running in a Web browser can make only limited use of the new capabilities. However, for server-to-server communication, HTTP/2 provides a lot of ways to enhance existing REST APIs.

This is the first in a series of blog posts exploring the implications of HTTP/2 on REST services for server-to-server communication. Along with this blog, I develop [h2c], a simple command-line HTTP/2 client that is used to illustrate the examples in this blog.

### TL;DR

In this post we will

  * Run an HTTP/2-enabled Java application server.
  * Issue a GET request with the [h2c] command line tool.
  * Find out that the application server maintains two independent states: An HTTP/2 connection state, and an HTTP session state.

### Wildfly 9.0.0.Beta1 with HTTP/2 Support

Currently, all major Java servlet containers are adding HTTP/2 support.
For the Hello, World! example we use Wildfly 9.0.0.Beta1.
In order to get it running, you can either follow the [instructions on the Undertow blog], or you can use a pre-built [Docker image] that I made for this purpose.

Wildfly's IP address depends on if you run it with or without Docker, and if you are using Docker on Linux, or [boot2docker] on another operating system. For simplicity, I will assume Wilfly is running on [https://192.168.59.103:8443].

As an [example application], I created a simple Java REST service, which can be downloaded and built follows:

{% highlight bash %}
git clone https://github.com/fstab/http2-examples
cd http2-examples/hello-world
mvn clean package
{% endhighlight %}

The resulting file `target/hello-world.war` can be deployed using Wildfly's management console on [http://192.168.59.103:9990/console]. The Docker image is pre-configured with username _admin_ and password _admin_.

The REST service can be called on [https://192.168.59.103:8443/hello-world/api/hello-world] with any HTTP/2 enabled Web browser, as for example Google's Chrome:

![REST service in Google Chrome]( {{site.url}}{{site.baseurl}}/assets/2015-06-05-rest-service-in-chrome.png)

The service responds _Hello, World!_, and shows how many times it has been called in the current HTTP session.

{% highlight java %}
@GET
@Produces(javax.ws.rs.core.MediaType.TEXT_PLAIN)
public String sayHello(@Context HttpServletRequest request) {
    AtomicInteger n = (AtomicInteger) request.getSession().getAttribute("n");
    if (n == null) {
        n = new AtomicInteger(0);
        request.getSession().setAttribute("n", n);
    }
    return "Hello, World!\n" +
            "Btw, this is request number " + n.incrementAndGet() + ".";
}
{% endhighlight %}

### HTTP/2 Client

In order to illustrate the examples in this blog series, I develop [h2c], a simple command-line HTTP/2 client.
The [h2c] tool is built using Brad Fitzpatrick's [HTTP/2 support for Go].
Currently, the [h2c] tool is an a very early stage: It can only do a simple "good case" GET request. 
More features will be added along with upcoming blog posts in this series.

However, we can use [h2c] to find out the relationship between Sessions and Connections.

[h2c] is a simple executable, it can be run without any dependencies. Download [h2c] for your operating system, and perform a GET request as follows:

{% highlight bash %}
h2c start &
h2c connect 192.168.59.103:8443
h2c get /hello-world/api/hello-world
{% endhighlight %}

If you are used to wget's

{% highlight bash %}
wget --no-certificate-check https://192.168.59.103:8443/hello-world/api/hello-world
{% endhighlight %}

you might be wondering why we need three commands here. The reason is that with traditional tools like `wget`, there is a 1-to-1 relationship between HTTP connections and HTTP requests. Each time you run `wget`, it opens a connection, performs the request, and closes the connection again.

`h2c start &` runs a background process that maintains long-running connections. A connection can be used for multiple requests, and it enables bi-directional communication.

For now, we can issue multiple GET requests in a single connection to find out if the request counter increases:

![REST service using h2c]( {{site.url}}{{site.baseurl}}/assets/2015-06-05-rest-service-in-h2c.png)

As you can see, the result is always '1'. That means that HTTP/2's connection state is independent of the application server's HTTP session.
We could couple these states using cookie headers, but we can also use them separately. We will look deeper into this in the upcoming blog posts.

### What's Next?

This post was mainly to say _Hello_, and to get things up and running. However, we learned an interesting fact already, which is that HTTP/2's connection state is independent of the servlet container's HTTP session.

The upcoming posts will dig deeper into what happens on the wire, and look into the more advanced features of HTTP/2. We will learn useful stuff, and we will learn things that are not-so-usefull-but-fun-to-expoit.

[HTTP/2]: https://http2.github.io/http2-spec
[h2c]:    https://github.com/fstab/h2c
[instructions on the Undertow blog]: http://undertow.io/blog/2015/03/26/HTTP2-In-Wildfly.html
[Docker image]: https://github.com/fstab/docker-wildfly-http2
[boot2docker]: http://boot2docker.io
[example application]: https://github.com/fstab/http2-examples/tree/master/hello-world
[http://192.168.59.103:9990/console]: http://192.168.59.103:9990/console
[https://192.168.59.103:8443]: https://192.168.59.103:8443
[https://192.168.59.103:8443/hello-world/api/hello-world]: https://192.168.59.103:8443/hello-world/api/hello-world
[HTTP/2 support for Go]: https://github.com/bradfitz/http2
