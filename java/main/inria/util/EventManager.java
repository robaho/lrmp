// EventManager.java
// $Id: EventManager.java,v 1.2 1996/05/28 15:50:40 abaird Exp $
// (c) COPYRIGHT MIT and INRIA, 1996.
// Please first read the full copyright statement in file COPYRIGHT.html

/**
 * A timer handling package.
 * This package was written by J Payne.
 */
package inria.util;

import java.util.Timer;
import java.util.TimerTask;


/**
 * This implements an event manager for timer events.  Timer events
 * are a way to have events occur some time in the future.  They are an
 * alternative to using separate threads which issue sleep requests
 * themselves.
 */
public class EventManager {

    static EventManager shared;
    static {
        shared = new EventManager();
    }

    static public EventManager shared() {
        return shared;
    }

    Timer timer = new Timer("Event Manager");

    private EventManager() {
    }

    /**
     * registerTimer inserts a new timer event into the queue.  The
     * queue is always sorted by time, in increasing order.  That is,
     * things farther into the future are further down in the queue.
     * ms is milliseconds in the future, handler is the object that
     * will handle the event, and data is a "rock" that is passed to
     * the handler to do with what it will.
     */
    public Event registerTimer(long ms, EventHandler handler, Object data) {
        long time = ms + System.currentTimeMillis();

        Event event = new Event(time, handler, data);
        timer.schedule(event,ms);
        return event;
    }

    /**
     * This recalls a previously registered timer event.
     */
    public void recallTimer(Event timer) {
        timer.cancel();
    }

    static public class Event extends TimerTask {

        /**
         * Absolute time, in ms, to deliver this event.
         */
        long time;

        /**
         * Piece of data to pass to the event handler.
         */
        Object data;

        /**
         * handler for this event
         */
        EventHandler handler;

        Event(long time, EventHandler handler, Object data) {
            this.time = time;
            this.handler = handler;
            this.data = data;
        }

        @Override
        public void run() {
            try {
                handler.handleTimerEvent(data, time);
            } catch(Exception e){
                Logger.error("unable to execute event",e);
            }
        }
    }

}

