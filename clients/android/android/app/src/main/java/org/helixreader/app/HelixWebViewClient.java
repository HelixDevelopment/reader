package org.helixreader.app;

import android.webkit.WebResourceRequest;
import android.webkit.WebResourceResponse;
import android.webkit.WebView;
import com.getcapacitor.Bridge;
import com.getcapacitor.BridgeWebViewClient;
import java.util.Map;

/** Capacitor's WebViewClient plus: Helix API reads go through Cronet (HTTP/3 + Brotli); everything else is Capacitor's. */
final class HelixWebViewClient extends BridgeWebViewClient {
    private final CronetTransport transport;

    HelixWebViewClient(Bridge bridge, CronetTransport transport) {
        super(bridge);
        this.transport = transport;
    }

    @Override
    public WebResourceResponse shouldInterceptRequest(WebView view, WebResourceRequest request) {
        String url = request.getUrl().toString();
        if (!ApiRequestPolicy.shouldHandle(request.getMethod(), url)) return super.shouldInterceptRequest(view, request);
        Map<String, String> h = request.getRequestHeaders();
        String origin = h.get("Origin");
        Map<String, String> cors = ApiRequestPolicy.corsHeaders(origin, h.get("Access-Control-Request-Headers"));
        try {
            return transport.serve(request.getMethod(), url, h, cors);
        } catch (RuntimeException e) {
            return null; // the WebView's own stack answers (never a silent hang)
        }
    }
}
