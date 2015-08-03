package de.consol.labs.h2c;

import javax.servlet.ServletException;
import javax.servlet.annotation.WebServlet;
import javax.servlet.http.HttpServlet;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;
import javax.servlet.http.HttpSession;
import java.io.BufferedReader;
import java.io.IOException;
import java.io.PrintWriter;
import java.util.concurrent.atomic.AtomicInteger;

@WebServlet("test")
public class TestServlet extends HttpServlet {

    // Mapped to https://localhost:8443/h2c/test
    // The size of the returned text can be increased with a URL parameter, like
    // https://localhost:8443/hello-world/api/hello-world?size=123
    @Override
    protected void doGet(final HttpServletRequest req, final HttpServletResponse resp) throws ServletException, IOException {
        HttpSession session = req.getSession();
        if (session.getAttribute("counter") == null) {
            session.setAttribute("counter", new AtomicInteger(0));
        }
        int reqNr = ((AtomicInteger) session.getAttribute("counter")).incrementAndGet();
        PrintWriter writer = resp.getWriter();
        writer.write("Hello, World!\n");
        writer.write("Btw, this is request number " + reqNr + ".\n");
        int size = readSize(req);
        if (size > 0) {
            writer.write("Here are " + size + " 'a' characters\n");
            for (int i = 1; i <= size; i++) {
                writer.write("a");
                if (i % 80 == 0) {
                    writer.write("\n");
                }
            }
        }
        writer.close();
    }

    private int readSize(HttpServletRequest request) {
        if (request.getParameterMap().containsKey("size")) {
            return Integer.parseInt(request.getParameter("size"));
        }
        return 0;
    }

    // Mapped to https://localhost:8443/h2c/test
    @Override
    protected void doPost(final HttpServletRequest req, final HttpServletResponse resp) throws ServletException, IOException {
        int i = 0;
        while (req.getReader().read() != -1) {
            i++;
        }
        PrintWriter writer = resp.getWriter();
        writer.write("Received " + i + " characters.\n");
        writer.close();
    }
}
