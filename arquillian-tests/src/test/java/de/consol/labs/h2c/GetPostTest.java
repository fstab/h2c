package de.consol.labs.h2c;

import de.consol.labs.h2c.client.H2c;
import de.consol.labs.h2c.client.H2cBackgroundProcess;
import org.jboss.arquillian.container.test.api.Deployment;
import org.jboss.arquillian.junit.Arquillian;
import org.jboss.shrinkwrap.api.ShrinkWrap;
import org.jboss.shrinkwrap.api.spec.WebArchive;
import org.junit.*;
import org.junit.runner.RunWith;

import java.io.File;
import java.io.FileOutputStream;
import java.io.IOException;

import static java.lang.String.format;
import static org.junit.Assert.assertTrue;

@RunWith(Arquillian.class)
public class GetPostTest {

    private H2cBackgroundProcess backgroundProcess;
    private String path = "/h2c/test"; // name of war deployment + servlet path
    private String ip;

    @Deployment(testable = false)
    public static WebArchive createDeployment() {
        return ShrinkWrap.create(WebArchive.class, "h2c.war")
                .addClasses(TestServlet.class);
    }

    @Before
    public void setUp() throws IOException, InterruptedException {
        Assert.assertNull(backgroundProcess); // Make sure tearDown() was run successfully after last test.
        ip = getBoot2DockerIP();
        backgroundProcess = H2cBackgroundProcess.start();
        H2c.runWithTimeout(format("connect %s:8443", ip), 1);
    }

    private String getBoot2DockerIP() {
        String dockerHost = System.getenv("DOCKER_HOST");
        if (dockerHost == null || dockerHost.startsWith("unix://")) {
            // running natively on Linux
            return "localhost";
        } else {
            // running with boot2docker
            return dockerHost.replaceFirst(".*://", "").replaceFirst(":[0-9]*", "");
        }
    }

    @After
    public void tearDown() throws InterruptedException, IOException {
        backgroundProcess.stop(1);
        backgroundProcess = null;
    }

    @Test
    public void testGet() throws IOException, InterruptedException {
        H2c result = H2c.runWithTimeout(format("get %s", path), 1);
        Assert.assertTrue(result.getStdout().contains("Btw, this is request number " + 1));
    }

    /**
     * The initial value for the flow-control window is 65,535 octets.
     * Get a few more bytes to see if flow control works.
     */
    @Test
    public void testGetFlowControl() throws IOException, InterruptedException {
        H2c result = H2c.runWithTimeout(format("get %s?size=66000", path), 1);
        Assert.assertTrue(result.getStdout().contains("Btw, this is request number " + 1));
        Assert.assertTrue(result.getStdout().length() > 66000);
    }

    private File makeTmpFile(int nBytes) throws IOException {
        File tmp = File.createTempFile("h2c-test-data-", ".dat");
        tmp.deleteOnExit();
        try (FileOutputStream s = new FileOutputStream(tmp)) {
            s.write(new byte[nBytes]);
        }
        return tmp;
    }

    @Test
    public void testPut() throws IOException, InterruptedException {
        File dataFile = makeTmpFile(24);
        H2c result = H2c.runWithTimeout(format("put --file %s %s", dataFile.getAbsolutePath(), path), 1);
        assertTrue("Unexpected output: " + result.getStdout(), result.getStdout().contains("Received put request with " + 24 + " characters payload."));
        assertTrue(dataFile.delete());
    }

    @Test
    public void testPost() throws IOException, InterruptedException {
        File dataFile = makeTmpFile(27);
        H2c result = H2c.runWithTimeout(format("post --file %s %s", dataFile.getAbsolutePath(), path), 1);
        assertTrue("Unexpected output: " + result.getStdout(), result.getStdout().contains("Received post request with " + 27 + " characters payload."));
        assertTrue(dataFile.delete());
    }

    /**
     * The initial value for the flow-control window is 65,535 octets.
     * Post a few more bytes to see if flow control works.
     */
    @Test
    public void testPostFlowControl() throws IOException, InterruptedException {
        File dataFile = makeTmpFile(65000);
        H2c result = H2c.runWithTimeout(format("post --file %s %s", dataFile.getAbsolutePath(), path), 1);
        assertTrue("Unexpected output: " + result.getStdout(), result.getStdout().contains("Received post request with " + 65000 + " characters payload."));
        assertTrue(dataFile.delete());
    }
}
