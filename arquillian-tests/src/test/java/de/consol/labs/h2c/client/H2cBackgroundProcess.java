package de.consol.labs.h2c.client;

import org.apache.commons.exec.CommandLine;
import org.apache.commons.exec.DefaultExecuteResultHandler;
import org.apache.commons.exec.DefaultExecutor;
import org.apache.commons.exec.PumpStreamHandler;
import org.junit.Assert;

import java.io.IOException;

import static java.util.concurrent.TimeUnit.SECONDS;

public class H2cBackgroundProcess {

    private OutputCapturingStream out = new OutputCapturingStream();
    private DefaultExecuteResultHandler resultHandler = new DefaultExecuteResultHandler();

    public static H2cBackgroundProcess start() throws IOException, InterruptedException {
        H2cBackgroundProcess process = new H2cBackgroundProcess();
        CommandLine cmdLine = CommandLine.parse("h2c start");
        DefaultExecutor executor = new DefaultExecutor();
        executor.setStreamHandler(new PumpStreamHandler(process.out));
        executor.execute(cmdLine, process.resultHandler);
        Thread.sleep(500);
        return process;
    }

    public void stop(int timeoutInSeconds) throws InterruptedException, IOException {
        H2c.runWithTimeout("stop", 1);
        resultHandler.waitFor(SECONDS.toMillis(timeoutInSeconds));
        Assert.assertTrue("Process should have terminated in " + timeoutInSeconds + " seconds.", resultHandler.hasResult());
        Assert.assertEquals("Process failed with exit code " + resultHandler.getExitValue() + ".\nOuput was:\n" + String.join("\n", out.getLines()) + "\n", 0, resultHandler.getExitValue());
    }
}
