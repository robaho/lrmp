/*
 * COPYRIGHT 1995 BY: MASSACHUSETTS INSTITUTE OF TECHNOLOGY (MIT), INRIA
 * 
 * This W3C software is being provided by the copyright holders under the
 * following license. By obtaining, using and/or copying this software, you
 * agree that you have read, understood, and will comply with the following
 * terms and conditions:
 * 
 * Permission to use, copy, modify, and distribute this software and its
 * documentation for any purpose and without fee or royalty is hereby granted,
 * provided that the full text of this NOTICE appears on ALL copies of the
 * software and documentation or portions thereof, including modifications,
 * that you make.
 * 
 * THIS SOFTWARE IS PROVIDED "AS IS," AND COPYRIGHT HOLDERS MAKE NO
 * REPRESENTATIONS OR WARRANTIES, EXPRESS OR IMPLIED. BY WAY OF EXAMPLE, BUT
 * NOT LIMITATION, COPYRIGHT HOLDERS MAKE NO REPRESENTATIONS OR WARRANTIES OF
 * MERCHANTABILITY OR FITNESS FOR ANY PARTICULAR PURPOSE OR THAT THE USE OF THE
 * SOFTWARE OR DOCUMENTATION WILL NOT INFRINGE ANY THIRD PARTY PATENTS,
 * COPYRIGHTS, TRADEMARKS OR OTHER RIGHTS. COPYRIGHT HOLDERS WILL BEAR NO
 * LIABILITY FOR ANY USE OF THIS SOFTWARE OR DOCUMENTATION.
 * 
 * The name and trademarks of copyright holders may NOT be used in advertising
 * or publicity pertaining to the software without specific, written prior
 * permission. Title to copyright in this software and any associated
 * documentation will at all times remain with copyright holders.
 */

/*
 * LrmpEntity.java - LRMP entity.
 * Author:  Tie Liao (Tie.Liao@inria.fr).
 * Created: 23 April 1997.
 * Updated: no.
 */
package inria.net.lrmp;

import java.net.InetAddress;

/**
 * encapsulates the information about an LRMP entity. This object is created and
 * managed internally by LRMP.
 */
public class LrmpEntity {
    private InetAddress ipAddr;     /* IP address */
    private long lastTimeHeard;
    private int nack;
    private int id;

    /**
     * round trip time in millis.
     */
    protected int rtt = 0;

    /**
     * approximative number of hops from local site.
     */
    protected int distance;

    /**
     * forbidden applications instantiate this object.
     */
    LrmpEntity(int id, InetAddress netaddr) {
        this.id = id;
        this.ipAddr = netaddr;

        reset();
    }

    protected void reset() {
        nack = 0;
        lastTimeHeard = 0;
        distance = 255;
    }

    protected void setAddress(InetAddress netaddr) {
        ipAddr = netaddr;
    }

    /**
     * Sets the entity ID.
     */
    protected void setID(int id) {
        this.id = id;
    }

    public int getID() {
        return id;
    }

    /**
     * Returns the network address.
     */
    public InetAddress getAddress() {
        return ipAddr;
    }

    /**
     * Returns a string representation of this object.
     */
    public String toString() {
        return Integer.toHexString(id) + "@" + ipAddr.getHostAddress();
    }

    /**
     * Returns the number of NACK packets heard from this entity.
     */
    public int getNackCount() {
        return nack;
    }

    /**
     * Returns the round trip time, in milliseconds.
     */
    public int getRTT() {
        return rtt;
    }

    protected void incNack() {
        nack++;
    }

    /**
     * Returns the last time when this entity was heard.
     */
    public long getLastTimeHeard() {
        return lastTimeHeard;
    }

    /**
     * Compares with an Lrmp entity for equality. Two entities are equal if they
     * have the same identifier.
     */
    public boolean equals(LrmpEntity e) {
        return (id == e.getID());
    }

    /*
     * Sets the last time when this entity was heard.
     */
    protected void setLastTimeHeard(long t) {
        lastTimeHeard = t;
    }
}

