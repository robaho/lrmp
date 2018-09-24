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
 * EntityTable.java - a table of entities indexed by ID.
 * Author:  Tie Liao (Tie.Liao@inria.fr).
 * Created: 20 May 1998.
 * Updated: no.
 */
package inria.util;

import inria.net.lrmp.LrmpEntity;

import java.util.HashMap;
import java.util.Iterator;
import java.util.Map;
import java.util.function.Predicate;

/**
 * EntityTable contains a set of entities indexed by ID.
 */
public class EntityTable implements Iterable<LrmpEntity> {
    private Map<Integer, LrmpEntity> map = new HashMap<>();

    /**
     * constructs an EntityTable object.
     */
    public EntityTable() {
    }

    /**
     * adds the given entity to the table.
     * @param obj the entity to be added.
     */
    public void addEntity(LrmpEntity obj) {
        map.put(obj.getID(),obj);
    }

    /**
     * gets the entity.
     * @param id the entity ID.
     */
    public LrmpEntity getEntity(int id) {
        return map.get(id);
    }

    /**
     * remove the given entity from the table.
     * @param obj the entity to remove.
     */
    public void removeEntity(LrmpEntity obj) {
        LrmpEntity e = map.remove(obj.getID());
        if (e!=null && e!=obj) {
            addEntity(obj);
        }
    }

    public int size() {
        return map.size();
    }

    /**
     * drops old recorded entities.
     */
    public void prune(Predicate<LrmpEntity> filter) {

        Iterator<LrmpEntity> i = map.values().iterator();
        while(i.hasNext()) {
            LrmpEntity e = i.next();
            if (filter.test(e)) {
                i.remove();
            }
        }
    }

    @Override
    public Iterator<LrmpEntity> iterator() {
        return map.values().iterator();
    }
}

