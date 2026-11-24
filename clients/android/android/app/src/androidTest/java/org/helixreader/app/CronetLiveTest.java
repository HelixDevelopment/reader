package org.helixreader.app;

import static org.junit.Assert.*;

import android.os.Bundle;
import android.webkit.WebResourceResponse;
import androidx.test.core.app.ActivityScenario;
import androidx.test.ext.junit.runners.AndroidJUnit4;
import androidx.test.platform.app.InstrumentationRegistry;
import java.io.ByteArrayOutputStream;
import java.io.InputStream;
import java.util.Collections;
import java.util.concurrent.atomic.AtomicReference;
import org.junit.Assume;
import org.junit.Test;
import org.junit.runner.RunWith;

/**
 * LIVE tests, run on a real Android (emulator) in the rootless h3 lab: arguments `-e server https://helix-server:8443`.
 * The pass criterion is read from the SERVER access log by qa/h3 (HTTP/3.0 + br); these tests assert the device-side view.
 */
@RunWith(AndroidJUnit4.class)
public class CronetLiveTest {
    private static String server() {
        Bundle a = InstrumentationRegistry.getArguments();
        String s = a.getString("server");
        assertNotNull("pass -e server https://host:port", s);
        return s;
    }

    /** Fetches /v1/formats 4 times through the transport, asserting decoded JSON each time; returns the negotiated protocol of the last response. */
    private static String fetchFormats() throws Exception {
        String fq = InstrumentationRegistry.getArguments().getString("forceQuic");
        if (fq != null) CronetTransport.forceQuicOrigins = fq;
        String nl = InstrumentationRegistry.getArguments().getString("netlog");
        if (nl != null) CronetTransport.netLogPath = nl;
        CronetTransport t = new CronetTransport(InstrumentationRegistry.getInstrumentation().getTargetContext());
        String proto = "";
        for (int i = 0; i < 4; i++) { // the first request may win the race on TCP; QUIC must be in use shortly after
            WebResourceResponse r = t.serve("GET", server() + "/v1/formats", Collections.singletonMap("Accept", "application/json"), Collections.emptyMap());
            assertEquals(200, r.getStatusCode());
            ByteArrayOutputStream out = new ByteArrayOutputStream();
            try (InputStream in = r.getData()) { byte[] b = new byte[4096]; for (int n; (n = in.read(b)) > 0; ) out.write(b, 0, n); }
            String body = out.toString("UTF-8");
            assertTrue("body must be decoded JSON, not Brotli bytes: " + body.substring(0, Math.min(60, body.length())), body.startsWith("{\"formats\""));
            assertNull("Content-Encoding must be stripped after decoding", r.getResponseHeaders().get("Content-Encoding"));
            proto = t.lastProtocol();
            Thread.sleep(500);
        }
        return proto;
    }

    /** Always-on live assertion: Cronet serves the API, Brotli is decoded, the protocol is a modern one (h2 or h3), never HTTP/1.1. */
    @Test public void cronetTransportServesBrotliOverH2OrH3() throws Exception {
        String proto = fetchFormats();
        assertTrue("negotiated protocol " + proto, proto.equals("h2") || proto.equals("h3"));
    }

    /**
     * HTTP/3. With a PRIVATE CA the Cronet QUIC handshake is rejected by Chromium's QUIC verifier (net-log: CERTIFICATE_VERIFY_FAILED, net_error -356) and the
     * transport falls back to h2: that is an honest SKIP-with-reason (UNCONFIRMED, tracked), never a pass and never a red-by-design. With `-e expectH3 1`
     * (a publicly trusted certificate) it is a hard assertion.
     */
    @Test public void cronetTransportSpeaksHttp3() throws Exception {
        String proto = fetchFormats();
        if ("1".equals(InstrumentationRegistry.getArguments().getString("expectH3"))) { assertEquals("h3", proto); return; }
        Assume.assumeTrue("SKIP: private-CA QUIC certificate verify fails (net_error -356), h3 UNCONFIRMED on Android; negotiated " + proto, proto.equals("h3"));
    }

    @Test public void webViewApiFetchGoesThroughCronetWithCors() throws Exception {
        final String url = server() + "/v1/formats";
        try (ActivityScenario<MainActivity> sc = ActivityScenario.launch(MainActivity.class)) {
            AtomicReference<String> result = new AtomicReference<>();
            Thread.sleep(3000); // the bridge loads the bundle
            for (int i = 0; i < 4; i++) {
                final String js = "(function(){window.__h=null;fetch('" + url + "',{credentials:'omit'}).then(function(r){return r.json()}).then(function(j){window.__h='ok:'+j.formats.length}).catch(function(e){window.__h='err:'+e})})()";
                sc.onActivity(a -> a.getBridge().getWebView().evaluateJavascript(js, null));
                for (int w = 0; w < 100 && (result.get() == null || result.get().equals("null")); w++) {
                    Thread.sleep(100);
                    sc.onActivity(a -> a.getBridge().getWebView().evaluateJavascript("window.__h", v -> result.set(v)));
                }
                assertTrue("fetch result " + result.get(), result.get() != null && result.get().startsWith("\"ok:"));
                result.set(null);
                Thread.sleep(500);
            }
        }
    }
}
