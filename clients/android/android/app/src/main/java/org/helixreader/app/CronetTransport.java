package org.helixreader.app;

import android.content.Context;
import android.util.Log;
import android.webkit.WebResourceResponse;
import java.io.ByteArrayInputStream;
import java.io.IOException;
import java.io.InputStream;
import java.net.URI;
import java.nio.ByteBuffer;
import java.nio.charset.StandardCharsets;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.LinkedBlockingQueue;
import java.util.concurrent.TimeUnit;
import org.chromium.net.CronetEngine;
import org.chromium.net.CronetException;
import org.chromium.net.ExperimentalCronetEngine;
import org.chromium.net.UrlRequest;
import org.chromium.net.UrlResponseInfo;

/**
 * Serves the Helix API requests of the WebView through Cronet (Chromium's network stack as a library): HTTP/3 over QUIC,
 * HTTP/2, Brotli. One engine per origin, each with a QUIC hint for that origin so the first request already races QUIC
 * (Alt-Svc discovery alone would need a first TCP response and a publicly trusted certificate). The response body streams.
 * The negotiated protocol of every response is logged (tag HelixCronet) as device-side evidence; the server's access log is the
 * authoritative one.
 */
final class CronetTransport {
    static final String TAG = "HelixCronet";
    private static final int HEAD_TIMEOUT_S = 60;
    private final Context context;
    private final Map<String, CronetEngine> engines = new ConcurrentHashMap<>();
    private volatile String lastProtocol = "";
    /** Lab/test only (never set in production): "host:port" origins Chromium must force QUIC on. A private CA is never upgraded by hint/Alt-Svc alone. */
    static volatile String forceQuicOrigins = "";
    /** Lab/test only: file path for a Cronet net-log (diagnostic of the QUIC decision). */
    static volatile String netLogPath = "";
    private final ExecutorService executor = Executors.newCachedThreadPool();

    CronetTransport(Context context) {
        this.context = context.getApplicationContext();
    }

    /** Negotiated protocol of the most recent response ("h3", "h2", "http/1.1"): test/diagnostic hook. */
    String lastProtocol() { return lastProtocol; }

    private CronetEngine engineFor(URI u) {
        int port = u.getPort() != -1 ? u.getPort() : (u.getScheme().equals("https") ? 443 : 80);
        String key = u.getScheme() + "://" + u.getHost().toLowerCase() + ":" + port;
        return engines.computeIfAbsent(key, k -> {
            ExperimentalCronetEngine.Builder b = new ExperimentalCronetEngine.Builder(context);
            b.enableHttp2(true).enableQuic(true).enableBrotli(true)
                    .enableHttpCache(CronetEngine.Builder.HTTP_CACHE_DISABLED, 0); // the web client caches; no shared storage path between engines
            if (u.getScheme().equals("https")) b.addQuicHint(u.getHost(), port, port);
            String force = forceQuicOrigins;
            if (force != null && force.matches("[A-Za-z0-9.:-]+")) b.setExperimentalOptions("{\"QUIC\":{\"origins_to_force_quic_on\":[\"" + force + "\"]}}");
            CronetEngine eng = b.build();
            String nl = netLogPath;
            if (nl != null && nl.startsWith("/data/")) eng.startNetLogToFile(nl, true);
            return eng;
        });
    }

    /** Returns the WebView response for {@code method url}; failures before the response head become a 502 JSON error, never a hang. */
    WebResourceResponse serve(String method, String url, Map<String, String> requestHeaders, Map<String, String> corsHeaders) {
        URI u = URI.create(url);
        Map<String, String> fwd = ApiRequestPolicy.forwardHeaders(requestHeaders);
        if (method.equalsIgnoreCase("OPTIONS")) { // CORS preflight is answered locally; the server has no CORS
            return new WebResourceResponse("text/plain", "utf-8", 204, "No Content", new LinkedHashMap<>(corsHeaders), new ByteArrayInputStream(new byte[0]));
        }
        Body body = new Body();
        CountDownLatch head = new CountDownLatch(1);
        final UrlResponseInfo[] info = new UrlResponseInfo[1];
        final Throwable[] failure = new Throwable[1];
        UrlRequest.Callback cb = new UrlRequest.Callback() {
            @Override public void onRedirectReceived(UrlRequest r, UrlResponseInfo i, String newUrl) {
                if (ApiRequestPolicy.shouldHandle("GET", newUrl) || newUrl.startsWith(u.getScheme() + "://" + u.getRawAuthority() + "/")) r.followRedirect();
                else r.cancel();
            }
            @Override public void onResponseStarted(UrlRequest r, UrlResponseInfo i) { info[0] = i; head.countDown(); }
            @Override public void onReadCompleted(UrlRequest r, UrlResponseInfo i, ByteBuffer buf) {
                buf.flip();
                byte[] chunk = new byte[buf.remaining()];
                buf.get(chunk);
                body.offer(chunk);
            }
            @Override public void onSucceeded(UrlRequest r, UrlResponseInfo i) { body.offer(Body.EOF); }
            @Override public void onFailed(UrlRequest r, UrlResponseInfo i, CronetException e) { failure[0] = e; head.countDown(); body.offer(e); }
            @Override public void onCanceled(UrlRequest r, UrlResponseInfo i) { failure[0] = new IOException("canceled"); head.countDown(); body.offer(Body.EOF); }
        };
        UrlRequest.Builder rb = engineFor(u).newUrlRequestBuilder(url, cb, executor).setHttpMethod(method.toUpperCase());
        for (Map.Entry<String, String> h : fwd.entrySet()) rb.addHeader(h.getKey(), h.getValue());
        UrlRequest req = rb.build();
        body.attach(req);
        req.start();
        try {
            if (!head.await(HEAD_TIMEOUT_S, TimeUnit.SECONDS)) { req.cancel(); return error(504, "Helix server timed out", corsHeaders); }
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            req.cancel();
            return error(502, "interrupted", corsHeaders);
        }
        if (info[0] == null) return error(502, "Cannot reach the Helix server: " + (failure[0] == null ? "unknown" : failure[0].getMessage()), corsHeaders);
        UrlResponseInfo i = info[0];
        lastProtocol = i.getNegotiatedProtocol();
        Log.i(TAG, "proto=" + i.getNegotiatedProtocol() + " status=" + i.getHttpStatusCode() + " cached=" + i.wasCached() + " url=" + u.getRawPath());
        int status = i.getHttpStatusCode();
        if (!ApiRequestPolicy.statusOk(status)) { req.cancel(); return error(502, "Unrepresentable upstream status " + status, corsHeaders); }
        Map<String, String> headers = ApiRequestPolicy.responseHeaders(i.getAllHeaders());
        headers.putAll(corsHeaders);
        String ct = i.getAllHeaders().containsKey("content-type") ? i.getAllHeaders().get("content-type").get(0) : (i.getAllHeaders().containsKey("Content-Type") ? i.getAllHeaders().get("Content-Type").get(0) : null);
        return new WebResourceResponse(ApiRequestPolicy.mimeOf(ct), ApiRequestPolicy.charsetOf(ct), status, ApiRequestPolicy.reason(status), headers, method.equalsIgnoreCase("HEAD") ? new ByteArrayInputStream(new byte[0]) : body);
    }

    private static WebResourceResponse error(int status, String message, Map<String, String> cors) {
        byte[] json = ("{\"error\":\"" + message.replace("\\", "\\\\").replace("\"", "'").replaceAll("[\\r\\n]", " ") + "\"}").getBytes(StandardCharsets.UTF_8);
        Map<String, String> h = new LinkedHashMap<>(cors);
        return new WebResourceResponse("application/json", "utf-8", status, ApiRequestPolicy.reason(status), h, new ByteArrayInputStream(json));
    }

    /** Pull-based body: a Cronet read is issued only when the WebView consumer needs more (back-pressure, bounded memory). */
    private static final class Body extends InputStream {
        static final Object EOF = new Object();
        private final LinkedBlockingQueue<Object> queue = new LinkedBlockingQueue<>();
        private UrlRequest request;
        private byte[] cur = new byte[0];
        private int pos;
        private boolean pending, done;

        void attach(UrlRequest r) { request = r; }
        void offer(Object o) { queue.add(o); }

        @Override public int read() throws IOException {
            byte[] one = new byte[1];
            int n = read(one, 0, 1);
            return n < 0 ? -1 : (one[0] & 0xff);
        }

        @Override public int read(byte[] b, int off, int len) throws IOException {
            if (len == 0) return 0;
            while (pos >= cur.length) {
                if (done) return -1;
                if (!pending) { pending = true; request.read(ByteBuffer.allocateDirect(32 * 1024)); }
                Object o;
                try {
                    o = queue.take();
                } catch (InterruptedException e) {
                    Thread.currentThread().interrupt();
                    throw new IOException("interrupted");
                }
                if (o == EOF) { done = true; return -1; }
                if (o instanceof Throwable) { done = true; throw new IOException("Helix server stream failed", (Throwable) o); }
                cur = (byte[]) o; pos = 0; pending = false;
            }
            int n = Math.min(len, cur.length - pos);
            System.arraycopy(cur, pos, b, off, n);
            pos += n;
            return n;
        }

        @Override public void close() { done = true; if (request != null) request.cancel(); }
    }
}
