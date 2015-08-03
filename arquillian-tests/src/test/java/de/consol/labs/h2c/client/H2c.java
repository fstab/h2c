package de.consol.labs.h2c.client;

import org.apache.commons.exec.CommandLine;
import org.apache.commons.exec.DefaultExecuteResultHandler;
import org.apache.commons.exec.DefaultExecutor;
import org.apache.commons.exec.PumpStreamHandler;
import org.junit.Assert;

import java.io.IOException;

import static java.util.concurrent.TimeUnit.SECONDS;

public class H2c {

    private final OutputCapturingStream stdout;
    private final OutputCapturingStream stderr;

    private H2c(OutputCapturingStream stdout, OutputCapturingStream stderr) {
        this.stdout = stdout;
        this.stderr = stderr;
    }

    public static H2c runWithTimeout(String params, int timeoutInSeconds) throws IOException, InterruptedException {
        DefaultExecutor executor = new DefaultExecutor();
        OutputCapturingStream stdout = new OutputCapturingStream();
        OutputCapturingStream stderr = new OutputCapturingStream();
        executor.setStreamHandler(new PumpStreamHandler(stdout, stderr));

        String cmd = "h2c " + params;
        DefaultExecuteResultHandler resultHandler = new DefaultExecuteResultHandler();
        System.out.println(cmd);
        executor.execute(CommandLine.parse(cmd), resultHandler);
        resultHandler.waitFor(SECONDS.toMillis(timeoutInSeconds));
        Assert.assertTrue(cmd + ": Timeout after " + timeoutInSeconds + " seconds", resultHandler.hasResult());
        Assert.assertEquals("Process failed with exit code " + resultHandler.getExitValue() + ". Dump of stderr:\n" + String.join("\n", stderr.getLines()) + "\n", 0, resultHandler.getExitValue());
        return new H2c(stdout, stderr);
    }

    public String getStdout() {
        return String.join("\n", stdout.getLines());
    }

    public String getStderr() {
        return String.join("\n", stderr.getLines());
    }
}
