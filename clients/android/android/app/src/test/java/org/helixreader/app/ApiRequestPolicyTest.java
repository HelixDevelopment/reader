package org.helixreader.app;

import static org.junit.Assert.*;

import java.util.*;
import org.junit.Test;

public class ApiRequestPolicyTest {
    @Test public void handlesCrossOriginApiGetOnly() {
        assertTrue(ApiRequestPolicy.shouldHandle("GET", "https://helix.lan:8443/v1/works?limit=1"));
        assertTrue(ApiRequestPolicy.shouldHandle("HEAD", "https://helix.lan/healthz"));
        assertTrue(ApiRequestPolicy.shouldHandle("OPTIONS", "https://helix.lan/v1/works"));
        assertTrue(ApiRequestPolicy.shouldHandle("get", "http://10.0.2.2:8080/readyz"));
    }
    @Test public void refusesEverythingElse() {
        assertFalse("app assets", ApiRequestPolicy.shouldHandle("GET", "https://localhost/index.html"));
        assertFalse("app origin api path", ApiRequestPolicy.shouldHandle("GET", "https://localhost/v1/works"));
        assertFalse("http localhost app", ApiRequestPolicy.shouldHandle("GET", "http://localhost/v1/works"));
        assertFalse("write method", ApiRequestPolicy.shouldHandle("POST", "https://helix.lan/v1/works"));
        assertFalse("PUT", ApiRequestPolicy.shouldHandle("PUT", "https://helix.lan/v1/works"));
        assertFalse("non api path", ApiRequestPolicy.shouldHandle("GET", "https://helix.lan/other"));
        assertFalse("prefix trick", ApiRequestPolicy.shouldHandle("GET", "https://helix.lan/v1x/works"));
        assertFalse("dot segments", ApiRequestPolicy.shouldHandle("GET", "https://helix.lan/v1/../admin"));
        assertFalse("encoded traversal", ApiRequestPolicy.shouldHandle("GET", "https://helix.lan/v1/%2e%2e/admin"));
        assertFalse("encoded slash", ApiRequestPolicy.shouldHandle("GET", "https://helix.lan/v1/a%2Fb"));
        assertFalse("ftp", ApiRequestPolicy.shouldHandle("GET", "ftp://helix.lan/v1/works"));
        assertFalse("garbage", ApiRequestPolicy.shouldHandle("GET", "::::"));
        assertFalse("null", ApiRequestPolicy.shouldHandle(null, null));
        assertFalse("userinfo", ApiRequestPolicy.shouldHandle("GET", "https://u:p@helix.lan/v1/works"));
    }
    @Test public void forwardsOnlyAllowListedHeadersCaseInsensitivelyAndDropsConditionals() {
        Map<String, String> in = new HashMap<>();
        in.put("Authorization", "Basic x"); in.put("accept", "application/json"); in.put("Range", "bytes=0-9");
        in.put("Cookie", "s=1"); in.put("Host", "evil"); in.put("If-None-Match", "\"e\""); in.put("If-Modified-Since", "x"); in.put("Origin", "https://localhost");
        Map<String, String> out = ApiRequestPolicy.forwardHeaders(in);
        assertEquals(new TreeSet<>(Arrays.asList("authorization", "accept", "range")), new TreeSet<>(out.keySet()));
        assertEquals("Basic x", out.get("authorization"));
    }
    @Test public void corsAnswersThePageOriginOnlyAndNeverWildcardWithCredentialsHeaders() {
        Map<String, String> h = ApiRequestPolicy.corsHeaders("https://localhost", "authorization, x-evil\r\nInjected: 1");
        assertEquals("https://localhost", h.get("Access-Control-Allow-Origin"));
        assertEquals("authorization", h.get("Access-Control-Allow-Headers"));
        assertEquals("Origin", h.get("Vary"));
        assertTrue(h.get("Access-Control-Expose-Headers").contains("Content-Range"));
        assertTrue(ApiRequestPolicy.corsHeaders(null, null).isEmpty());
        assertTrue("a foreign page origin gets no CORS", ApiRequestPolicy.corsHeaders("https://evil.example", null).isEmpty());
    }
    @Test public void upstreamHeadersDropEncodingLengthCookiesAndAuthChallenges() {
        Map<String, List<String>> up = new LinkedHashMap<>();
        up.put("Content-Encoding", Arrays.asList("br")); up.put("Content-Length", Arrays.asList("10")); up.put("Set-Cookie", Arrays.asList("a=b"));
        up.put("WWW-Authenticate", Arrays.asList("Basic")); up.put("Alt-Svc", Arrays.asList("h3=\":8443\"")); up.put("Content-Type", Arrays.asList("application/json; charset=utf-8"));
        up.put("Content-Range", Arrays.asList("bytes 0-9/100")); up.put("Access-Control-Allow-Origin", Arrays.asList("*"));
        Map<String, String> out = ApiRequestPolicy.responseHeaders(up);
        assertEquals(new TreeSet<>(Arrays.asList("Content-Type", "Content-Range")), new TreeSet<>(out.keySet()));
    }
    @Test public void mimeAndCharsetParsing() {
        assertEquals("application/json", ApiRequestPolicy.mimeOf("application/json; charset=utf-8"));
        assertEquals("utf-8", ApiRequestPolicy.charsetOf("application/json; charset=utf-8"));
        assertNull(ApiRequestPolicy.charsetOf("image/png"));
        assertEquals("application/octet-stream", ApiRequestPolicy.mimeOf(null));
    }
    @Test public void reasonPhraseIsNeverEmptyAndRedirectsAreMapped() {
        assertEquals("OK", ApiRequestPolicy.reason(200)); assertFalse(ApiRequestPolicy.reason(418).isEmpty());
        assertFalse(ApiRequestPolicy.statusOk(304)); assertTrue(ApiRequestPolicy.statusOk(206)); assertFalse(ApiRequestPolicy.statusOk(99));
    }
}
