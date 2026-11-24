package org.helixreader.app;

import android.os.Bundle;
import com.getcapacitor.BridgeActivity;

public class MainActivity extends BridgeActivity {
    @Override
    public void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        // HTTP/3 + Brotli for the Helix API through Cronet (Chromium network stack as a library).
        getBridge().setWebViewClient(new HelixWebViewClient(getBridge(), new CronetTransport(this)));
    }
}
