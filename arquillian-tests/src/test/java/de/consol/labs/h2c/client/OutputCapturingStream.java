package de.consol.labs.h2c.client;

import org.apache.commons.exec.LogOutputStream;

import java.util.LinkedList;
import java.util.List;

public class OutputCapturingStream extends LogOutputStream {

    private final List<String> lines = new LinkedList<>();

    @Override
    protected void processLine(String line, int level) {
        lines.add(line);
    }

    public List<String> getLines() {
        return lines;
    }
}
