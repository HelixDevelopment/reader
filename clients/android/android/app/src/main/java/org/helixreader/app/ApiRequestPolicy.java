package org.helixreader.app;

import java.net.URI;
import java.util.Arrays;
import java.util.HashSet;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Set;
import java.util.regex.Pattern;

/**
 * Pure (Android-free, JVM-testable) policy of the Cronet transport: WHICH WebView requests are served by Cronet
 * (HTTP/3 + Brotli) and how headers cross the boundary. Mirrors the desktop shell's lib/proxy.js: read-only methods,
 * the Helix API prefixes only, an allow-list of forwarded request headers, no cookies, no conditional requests
 * (a 304 cannot be represented by WebResourceResponse).
 */
final class ApiRequestPolicy {
    private ApiRequestPolicy() {}

    /** The Capacitor app origin (https://localhost, no port) is the page itself, never the API. */
    static final String APP_ORIGIN = "https://localhost";
    private static final Set<String> METHODS = new HashSet<>(Arrays.asList("GET", "HEAD", "OPTIONS"));
    private static final Set<String> FORWARD = new HashSet<>(Arrays.asList("accept", "accept-language", "range", "authorization"));
    private static final Set<String> DROP_RESPONSE = new HashSet<>(Arrays.asList(
            "set-cookie", "set-cookie2", "www-authenticate", "proxy-authenticate", "connection", "keep-alive", "transfer-encoding",
            "content-encoding", "content-length", "alt-svc", "access-control-allow-origin", "access-control-allow-credentials",
            "access-control-allow-headers", "access-control-allow-methods", "access-control-expose-headers", "access-control-max-age", "vary"));
    private static final Pattern HEADER_NAME = Pattern.compile("^[A-Za-z0-9-]{1,64}$");
    private static final Pattern BAD_PATH = Pattern.compile("(?i)(^|/)\\.\\.?(/|$)|%2e|%2f|%5c|\\\\");

    static boolean shouldHandle(String method, String url) {
        if (method == null || url == null || !METHODS.contains(method.toUpperCase(Locale.ROOT))) return false;
        URI u;
        try {
            u = URI.create(url);
        } catch (IllegalArgumentException e) {
            return false;
        }
        String scheme = u.getScheme();
        if (scheme == null || !(scheme.equals("https") || scheme.equals("http"))) return false;
        if (u.getRawUserInfo() != null || u.getHost() == null) return false;
        if ("localhost".equalsIgnoreCase(u.getHost()) && u.getPort() == -1) return false; // the app's own origin
        String path = u.getRawPath();
        if (path == null || BAD_PATH.matcher(path).find()) return false;
        return path.startsWith("/v1/") || path.equals("/healthz") || path.equals("/readyz");
    }

    /** Allow-listed request headers, lower-cased. Cookie, Host, Origin, conditionals are never forwarded. */
    static Map<String, String> forwardHeaders(Map<String, String> in) {
        Map<String, String> out = new LinkedHashMap<>();
        if (in == null) return out;
        for (Map.Entry<String, String> e : in.entrySet()) {
            String k = e.getKey() == null ? "" : e.getKey().toLowerCase(Locale.ROOT);
            if (FORWARD.contains(k) && e.getValue() != null && e.getValue().indexOf('\r') < 0 && e.getValue().indexOf('\n') < 0) out.put(k, e.getValue());
        }
        return out;
    }

    /** CORS answer for the page (the Helix server sends none). Only the app origin is answered; never a wildcard. */
    static Map<String, String> corsHeaders(String origin, String requestedHeaders) {
        Map<String, String> h = new LinkedHashMap<>();
        if (origin == null || !(origin.equals(APP_ORIGIN) || origin.equals("http://localhost"))) return h;
        h.put("Access-Control-Allow-Origin", origin);
        h.put("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS");
        StringBuilder allow = new StringBuilder();
        if (requestedHeaders != null) {
            for (String part : requestedHeaders.split(",")) {
                String n = part.trim().toLowerCase(Locale.ROOT);
                if (HEADER_NAME.matcher(n).matches() && (FORWARD.contains(n) || n.equals("content-type"))) {
                    if (allow.length() > 0) allow.append(", ");
                    allow.append(n);
                }
            }
        }
        if (allow.length() > 0) h.put("Access-Control-Allow-Headers", allow.toString());
        h.put("Access-Control-Expose-Headers", "Content-Length, Content-Range, Content-Type, ETag, X-Request-ID");
        h.put("Access-Control-Max-Age", "600");
        h.put("Vary", "Origin");
        return h;
    }

    /** Upstream response headers safe to hand to the WebView (Cronet already decoded the body: no Content-Encoding/Length). */
    static Map<String, String> responseHeaders(Map<String, List<String>> upstream) {
        Map<String, String> out = new LinkedHashMap<>();
        if (upstream == null) return out;
        for (Map.Entry<String, List<String>> e : upstream.entrySet()) {
            if (e.getKey() == null || e.getValue() == null || e.getValue().isEmpty()) continue;
            if (DROP_RESPONSE.contains(e.getKey().toLowerCase(Locale.ROOT))) continue;
            out.put(e.getKey(), String.join(", ", e.getValue()));
        }
        return out;
    }

    static String mimeOf(String contentType) {
        if (contentType == null || contentType.trim().isEmpty()) return "application/octet-stream";
        return contentType.split(";")[0].trim().toLowerCase(Locale.ROOT);
    }

    static String charsetOf(String contentType) {
        if (contentType == null) return null;
        for (String p : contentType.split(";")) {
            String t = p.trim();
            if (t.toLowerCase(Locale.ROOT).startsWith("charset=")) return t.substring(8).trim().replace("\"", "");
        }
        return null;
    }

    static boolean statusOk(int code) {
        return code >= 100 && code <= 599 && !(code >= 300 && code <= 399);
    }

    static String reason(int code) {
        switch (code) {
            case 200: return "OK";
            case 204: return "No Content";
            case 206: return "Partial Content";
            case 400: return "Bad Request";
            case 401: return "Unauthorized";
            case 403: return "Forbidden";
            case 404: return "Not Found";
            case 416: return "Range Not Satisfiable";
            case 500: return "Internal Server Error";
            case 502: return "Bad Gateway";
            case 503: return "Service Unavailable";
            default: return code >= 400 ? "Error" : "Status";
        }
    }
}
