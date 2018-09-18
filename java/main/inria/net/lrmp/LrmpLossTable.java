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
 * LrmpLossTable.java - table of loss packets.
 * Author:  Tie Liao (Tie.Liao@inria.fr).
 * Created: 30 May 1997.
 * Updated: no.
 */
package inria.net.lrmp;


import java.util.Iterator;
import java.util.LinkedList;
import java.util.List;

/**
 * table of loss packets.
 */
final class LrmpLossTable implements Iterable<LrmpLossEvent>{
    private List<LrmpLossEvent> table;

    public LrmpLossTable(int initialSize) {
        table = new LinkedList();
    }

    public void clear() {
        table.clear();
    }

    public int size() {
        return table.size();
    }

    public void add(LrmpLossEvent ev) {
        table.add(ev);
    }

    public void remove(LrmpLossEvent ev) {
        table.remove(ev);
    }

    public LrmpLossEvent lookup(LrmpSender s, LrmpEntity reporter) {
        for (LrmpLossEvent e : table){
            if (e.source==s && e.reporter==reporter) {
                return e;
            }
        }
        return null;
    }

    @Override
    public Iterator<LrmpLossEvent> iterator() {
        return table.iterator();
    }
}

