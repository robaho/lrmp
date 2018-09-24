/**
 * *******************************************************************
 * This Software is copyright INRIA. 1997.
 * 
 * INRIA holds all the ownership rights on the Software. The scientific
 * community is asked to use the SOFTWARE in order to test and evaluate
 * it.
 * 
 * INRIA freely grants the right to use the Software. Any use or
 * reproduction of this Software to obtain profit or for commercial ends
 * being subject to obtaining the prior express authorization of INRIA.
 * 
 * INRIA authorizes any reproduction of this Software
 * 
 * - in limits defined in clauses 9 and 10 of the Berne agreement for
 * the protection of literary and artistic works respectively specify in
 * their paragraphs 2 and 3 authorizing only the reproduction and quoting
 * of works on the condition that :
 * 
 * - "this reproduction does not adversely affect the normal
 * exploitation of the work or cause any unjustified prejudice to the
 * legitimate interests of the author".
 * 
 * - that the quotations given by way of illustration and/or tuition
 * conform to the proper uses and that it mentions the source and name of
 * the author if this name features in the source",
 * 
 * - under the condition that this file is included with any
 * reproduction.
 * 
 * Any commercial use made without obtaining the prior express agreement
 * of INRIA would therefore constitute a fraudulent imitation.
 * 
 * The Software beeing currently developed, INRIA is assuming no
 * liability, and should not be responsible, in any manner or any case,
 * for any direct or indirect dammages sustained by the user.
 * ******************************************************************
 */

/*
 * Utilities.java - Common utilities.
 * Author:  Tie Liao (Tie.Liao@inria.fr).
 * Created: 3 July 1996.
 * Updated: no.
 */
package inria.util;

import java.util.*;
import java.net.*;

/**
 * Static methods providing common utilities.
 */
public class Utilities {
    protected static InetAddress localhost = null;

    /**
     * gets the local host address.
     */
    public static InetAddress getLocalHost() {
        if (localhost == null) {
            try {
                localhost = InetAddress.getLocalHost();
            } catch (UnknownHostException e) {
                try {
                    localhost = InetAddress.getByName("127.0.0.1");
                } catch (UnknownHostException e1) {
                    return null;
                }
            }

            /* reverse lookup to get fully qualified host name */

            if (localhost.getHostName().indexOf('.') < 0) {
                try {
                    localhost = 
                        InetAddress.getByName(localhost.getHostAddress());
                } catch (UnknownHostException e) {}
            }
        }

        return localhost;
    }

    /**
     * Put an integer value into a byte array in the MSBF order.
     * @param i the integer value.
     * @param buff the byte array.
     * @param offset the offset in the array to put the integer.
     */
    public static void intToByte(int i, byte[] buff, int offset) {
        buff[offset] = (byte) (i >> 24);
        buff[offset + 1] = (byte) (i >> 16);
        buff[offset + 2] = (byte) (i >> 8);
        buff[offset + 3] = (byte) i;
    }

    /**
     * Get int32 value from a byte array.
     * @param buff the byte array.
     * @param offset the offset in the array.
     */
    public static int byteToInt(byte[] buff, int offset) {
        int i;

        i = ((buff[offset] << 24) & 0xff000000);
        i |= ((buff[offset + 1] << 16) & 0xff0000);
        i |= ((buff[offset + 2] << 8) & 0xff00);
        i |= (buff[offset + 3] & 0xff);

        return i;
    }

    /**
     * Get int16 value from a byte array.
     * @param buff the byte array.
     * @param offset the offset in the array.
     */
    public static int byteToShort(byte[] buff, int offset) {

        return ((buff[offset] & 0xFF) << 8) | (buff[offset+1] & 0xFF);
    }


    /**
     * Get int64 value from a byte array.
     * @param buff the byte array.
     * @param offset the offset in the array.
     */
    public static long byteToLong(byte[] buff, int offset) {
        long l;

        l = (buff[offset] << 56) & 0xff00000000000000L;
        l |= ((buff[offset + 1] << 48) & 0xff000000000000L);
        l |= ((buff[offset + 2] << 40) & 0xff0000000000L);
        l |= ((buff[offset + 3] << 32) & 0xff00000000L);
        l |= ((buff[offset + 4] << 24) & 0xff000000L);
        l |= ((buff[offset + 5] << 16) & 0xff0000L);
        l |= ((buff[offset + 6] << 8) & 0xff00L);
        l |= (buff[offset + 7] & 0xffL);

        return l;
    }

    static Random rand = null;

    /**
     * returns a random integer using the default seed.
     */
    public static int getRandomInteger() {
        if (rand == null) {
            rand = new Random();
        } 

        return rand.nextInt();
    }
}

