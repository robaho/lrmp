import inria.net.lrmp.*;
import inria.util.Logger;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.io.Reader;

public class LrmpTest {
    public static void main(String[] args) throws LrmpException, IOException {
        Logger.setDebug();

        LrmpProfile profile = new LrmpProfile();
        profile.setEventHandler(new LrmpEventHandler() {
            @Override
            public void processData(LrmpPacket pack) {
                System.out.println("I received a packet, with "+
                        new String(pack.getDataBuffer(),pack.getOffset(),pack.getDataLength()));
            }

            @Override
            public void processEvent(int event, Object data) {
                System.out.println("got an event "+data);
            }
        });
        Lrmp lrmp = new Lrmp("225.0.0.100",6000,0,"en0",profile);
        lrmp.start();

        Reader r = new BufferedReader(new InputStreamReader(System.in));
        String s;

        while ((s = ((BufferedReader) r).readLine())!=null) {
            byte[] bytes = s.getBytes();
            LrmpPacket p = new LrmpPacket(true,bytes.length);
            System.arraycopy(bytes,0,p.getDataBuffer(),p.getOffset(),bytes.length);
            p.setDataLength(bytes.length);
            System.out.println("sending packet");
            for(int i =0;i<100;i++){
                lrmp.send(p);
            }
        }
    }
}
