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
 * LrmpPacketQueue.java - ordered queue of lrmp packets.
 * Author:  Tie Liao (Tie.Liao@inria.fr).
 * Created: 16 June 1998.
 * Updated: no.
 */
package inria.net.lrmp;

import inria.util.Logger;

import java.util.*;

/**
 * ordered queue of lrmp packets. Sequence number may not be unique for
 * resend queue due to many sources in case of local recovery.
 */
final class LrmpPacketQueue {
    private Queue<LrmpPacket> queue = new PriorityQueue<>(new Comparator<LrmpPacket>() {
        @Override
        public int compare(LrmpPacket o1, LrmpPacket o2) {
            return Long.compare(o1.seqno,o2.seqno);
        }
    });

    /**
     * constructs an LrmpPacketQueue.
     */
    public LrmpPacketQueue(){}

    public boolean isEmpty() {
        return queue.isEmpty();
    }

    /**
     * adds the given packet to the queue.
     * @param obj the packet to be added.
     */
    public void enqueue(LrmpPacket obj) {
        queue.add(obj);
    }

    public boolean contains(LrmpPacket pack) {
        return queue.contains(pack);
    }

    /**
     * gets the packet.
     */
    public LrmpPacket dequeue() {
        return queue.poll();
    }

    /**
     * remove the packet with the given seqno from the queue.
     * @param seqno the packet seqno.
     */
    public void remove(LrmpSender s, long seqno, int scope) {
        Iterator<LrmpPacket> i = queue.iterator();
        while(i.hasNext()) {
            LrmpPacket p = i.next();
            if (s == p.sender && seqno == p.seqno && scope == p.scope) {
                i.remove();
                break;
            }
        }
    }

    public void cancel(LrmpSender s, int id, int scope) {
        Iterator<LrmpPacket> i = queue.iterator();
        while(i.hasNext()) {
            LrmpPacket p = i.next();
            if (s == p.sender && id == p.retransmitID && scope == p.scope) {
                i.remove();
                if (Logger.debug()) {
                    Logger.debug(this,
                            "cancel resend " + p.seqno + " " + queue.size());
                }
                break;
            }
        }
    }

}

