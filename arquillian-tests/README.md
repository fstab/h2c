h2c Arquillian Tests
--------------------

Test the [HTTP/2 client (h2c)](https://github.com/fstab/h2c) against
[Wildfly with HTTP/2 Support in a Docker Image](https://github.com/fstab/docker-wildfly-http2).

Run with [maven](https://maven.apache.org/) as follows:

```bash
mvn clean package
```

The tests use the [Arquillian Cube Extension](https://github.com/arquillian/arquillian-cube/)
to manage the [Docker](https://www.docker.com) containers.

There are some limitations with the current Arquillian Cube version 1.0.0.Alpha7:

  * The tests run only with [boot2docker](http://boot2docker.io), not on native Linux hosts.
  * There must be an entry in the [hosts file](https://en.wikipedia.org/wiki/Hosts_(file))
    mapping the hostname _boot2docker_ to the boot2docker ip.
  * boot2docker must run on port 2376.
