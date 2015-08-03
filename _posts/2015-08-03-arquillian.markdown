---
layout:   post
title:    "Testing HTTP/2 with Arquillian"
date:     2015-08-03 17:41:47
comments: true
---

Testing HTTP/2 communication requires a server and client communicating over TCP
connections.

While looking for a way to implement automated tests for the HTTP/2 client [h2c],
I came across [Arquillian Cube], which is an [Arquillian] extension that can be used
to manage [Docker] containers from Arquillian.

I created some initial tests in the [arquillian-tests] folder in the [h2c] GitHub
repository. The test can be run with [maven]:

{% highlight bash %}
mvn clean package
{% endhighlight %}

### TL;DR

This blog shows how to implement automated tests of HTTP/2 communication with 
[Arquillian Cube].

Running the tests in the [arquillian-tests] directory will:

* Start a [Wildfly HTTP/2 Docker container].
* Deploy a test Servlet.
* Run [h2c] GET and POST requests and verify the responses.

### How to Use Docker with Arquillian

First of all, the [arquillian-cube-docker] dependency need to be added to `pom.xml`
(in addition to the usual arquillian dependencies):

{% highlight xml %}
<dependency>
    <groupId>org.arquillian.cube</groupId>
    <artifactId>arquillian-cube-docker</artifactId>
    <version>1.0.0.Alpha7</version>
    <scope>test</scope>
</dependency>
{% endhighlight %}

Secondly, the [maven-surefire-plugin] must be configured to launch the `wildfly-docker`
container:

{% highlight xml %}
<plugin>
    <groupId>org.apache.maven.plugins</groupId>
    <artifactId>maven-surefire-plugin</artifactId>
    <version>2.17</version>
    <configuration>
        <systemPropertyVariables>
            <!-- The wildfly-docker container is defined in src/test/resources/arquillian.xml -->
            <arquillian.launch>wildfly-docker</arquillian.launch>
        </systemPropertyVariables>
    </configuration>
</plugin>
{% endhighlight %}

The container itself is configured in `arquillian.xml` in `src/test/resources`:

{% highlight xml %}
<?xml version="1.0"?>
<arquillian xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
            xmlns="http://jboss.org/schema/arquillian"
            xsi:schemaLocation="http://jboss.org/schema/arquillian
  http://jboss.org/schema/arquillian/arquillian_1_0.xsd">

    <extension qualifier="docker">

        <!-- The magic string dockerHost will be replaced with the boot2docker ip. -->
        <property name="serverUri">https://dockerHost:2376</property>

        <!-- The YAML configuration does not have any dynamic way to define the boot2docker ip. -->
        <!-- Therefore, we must add an entry in /etc/hosts to resolve boot2docker to boot2docker's ip -->
        <!-- See https://github.com/arquillian/arquillian-cube/issues/164 -->
        <property name="dockerContainers">
            wildfly-docker:
                image: fstab/wildfly-http2:9.0.0.Beta1
                await:
                    strategy: static
                    ip: boot2docker
                    ports: [8080, 8443, 9990]
                    sleepPollingTime: 1000
                    iterations: 120
                portBindings: ["8080", "9990", "8443"]
        </property>
    </extension>

    <!-- The container configuration uses the magic string dockerServerIp to point to the boot2docker ip. -->
    <container qualifier="wildfly-docker" default="true">
        <configuration>
            <property name="managementAddress">dockerServerIp</property>
            <property name="managementPort">9990</property>
            <property name="username">admin</property>
            <property name="password">admin</property>
        </configuration>
    </container>
</arquillian>
{% endhighlight %}

There are still some limitations with the current implementation:

* The tests run only with [boot2docker], not on native Linux hosts.
* There must be an entry in the [hosts file] mapping the host name _boot2docker_
  to the boot2docker ip.
* boot2docker must run on port 2376.

However, despite these limitations, [Arquillian Cube] provides a convenient way to test
HTTP/2 services through real TCP connections without using mocks.

[h2c]: https://github.com/fstab/h2c
[Arquillian Cube]: https://github.com/arquillian/arquillian-cube
[Arquillian]: http://arquillian.org
[Docker]: https://www.docker.com
[arquillian-tests]: https://github.com/fstab/h2c/tree/master/arquillian-tests
[maven]: https://maven.apache.org/
[Wildfly HTTP/2 Docker container]: https://registry.hub.docker.com/u/fstab/wildfly-http2
[arquillian-cube-docker]: https://github.com/arquillian/arquillian-cube
[maven-surefire-plugin]: https://maven.apache.org/surefire/maven-surefire-plugin/
[boot2docker]: http://boot2docker.io
[hosts file]: https://en.wikipedia.org/wiki/Hosts_(file)
