/**
 * *******************************************************************
 * This Software is copyright INRIA. 1997.
 * <p>
 * INRIA holds all the ownership rights on the Software. The scientific
 * community is asked to use the SOFTWARE in order to test and evaluate
 * it.
 * <p>
 * INRIA freely grants the right to use the Software. Any use or
 * reproduction of this Software to obtain profit or for commercial ends
 * being subject to obtaining the prior express authorization of INRIA.
 * <p>
 * INRIA authorizes any reproduction of this Software
 * <p>
 * - in limits defined in clauses 9 and 10 of the Berne agreement for
 * the protection of literary and artistic works respectively specify in
 * their paragraphs 2 and 3 authorizing only the reproduction and quoting
 * of works on the condition that :
 * <p>
 * - "this reproduction does not adversely affect the normal
 * exploitation of the work or cause any unjustified prejudice to the
 * legitimate interests of the author".
 * <p>
 * - that the quotations given by way of illustration and/or tuition
 * conform to the proper uses and that it mentions the source and name of
 * the author if this name features in the source",
 * <p>
 * - under the condition that this file is included with any
 * reproduction.
 * <p>
 * Any commercial use made without obtaining the prior express agreement
 * of INRIA would therefore constitute a fraudulent imitation.
 * <p>
 * The Software beeing currently developed, INRIA is assuming no
 * liability, and should not be responsible, in any manner or any case,
 * for any direct or indirect dammages sustained by the user.
 * ******************************************************************
 */

/*
 * Logger.java - Logger class.
 * Author:  Tie Liao (Tie.Liao@inria.fr).
 * Created: 6 June 1996.
 * Updated: no.
 */
package inria.util;

import sun.util.logging.LoggingSupport;

import java.io.PrintWriter;
import java.io.StringWriter;
import java.text.SimpleDateFormat;
import java.util.Date;
import java.util.logging.*;

/**
 * delegates to java util logging with a logger of 'lrmp'
 */
public class Logger {

    static class MyFormatter extends Formatter {
        // format string for printing the log record
        private static final String format = "[%1$s] %4$s: %5$s%n";
        private final Date dat = new Date();
        private final SimpleDateFormat df = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss:SS");

        public synchronized String format(LogRecord record) {
            dat.setTime(record.getMillis());
            String source;
            if (record.getSourceClassName() != null) {
                source = record.getSourceClassName();
                if (record.getSourceMethodName() != null) {
                    source += " " + record.getSourceMethodName();
                }
            } else {
                source = record.getLoggerName();
            }
            String message = formatMessage(record);
            String throwable = "";
            if (record.getThrown() != null) {
                StringWriter sw = new StringWriter();
                PrintWriter pw = new PrintWriter(sw);
                pw.println();
                record.getThrown().printStackTrace(pw);
                pw.close();
                throwable = sw.toString();
            }
            return String.format(format,
                    df.format(dat),
                    source,
                    record.getLoggerName(),
                    record.getLevel().getLocalizedName(),
                    message,
                    throwable);
        }
    }

    private static java.util.logging.Logger log = java.util.logging.Logger.getLogger("lrmp");
    private static Handler console = new ConsoleHandler();

    static {
        console.setFormatter(new MyFormatter());
        log.addHandler(console);
    }

    /**
     * turns on debug mode.
     */
    public static void setDebug() {
        log.setLevel(Level.FINE);
        console.setLevel(Level.FINE);
    }

    public static boolean debug() {
        return log.getLevel().intValue() >= Level.FINE.intValue();
    }

    public static boolean trace() {
        return log.getLevel().intValue() >= Level.FINEST.intValue();
    }

    /**
     * turns on/off the trace mode.
     * @param f the trace flag.
     */
    public static void setTrace() {
        log.setLevel(Level.FINEST);
    }

    /**
     * prints a trace message. If the error log file is set, this method always prints the
     * message to this file.
     * @param o the object from that the message is issued.
     * @param s the message to print.
     */
    public static void error(Object o, String s) {
        log.severe(o.getClass().getName() + ": " + s);
    }

    public static void error(String s) {
        log.severe(s);
    }

    public static void error(Object o, String s, Exception e) {
        error(o, s + " - " + e.getMessage());
    }

    public static void error(String s, Exception e) {
        error("error: " + s + " - " + e.getMessage());
    }

    public static void warning(Object o, String s) {
        log.warning(o.getClass().getName() + ": " + s);
    }

    public static void warning(String s) {
        log.warning(s);
    }

    /**
     * prints a message to stdout if the debug flag is true.
     * @param s the message to print.
     */
    public static void debug(String s) {
        log.fine(s);
    }

    /**
     * prints a message to stdout if the debug flag is true.
     * @param o the object from that the message is issued.
     * @param s the message to print.
     */
    public static void debug(Object o, String s) {
        log.fine(o.getClass().getName() + ": " + s);
    }

    /**
     * prints a message to stdout or the redirected logger if the trace flag is true.
     * @param s the message to print.
     */
    public static void trace(String s) {
        log.finest(s);
    }

    /**
     * prints a message to stdout or the redirected logger if the trace flag is true.
     * @param o the object from that the message is issued.
     * @param s the message to print.
     */
    public static void trace(Object o, String s) {
        log.finest(o.getClass().getName() + ": " + s);
    }

}
