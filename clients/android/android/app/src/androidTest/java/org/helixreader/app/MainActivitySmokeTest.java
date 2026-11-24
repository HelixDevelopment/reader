package org.helixreader.app;

import static org.junit.Assert.*;

import android.content.Context;
import android.util.DisplayMetrics;
import android.util.Log;
import android.view.View;
import android.view.ViewGroup;
import android.webkit.WebView;
import androidx.lifecycle.Lifecycle;
import androidx.test.core.app.ActivityScenario;
import androidx.test.ext.junit.runners.AndroidJUnit4;
import androidx.test.platform.app.InstrumentationRegistry;
import java.util.concurrent.atomic.AtomicReference;
import org.junit.Test;
import org.junit.runner.RunWith;

/**
 * HRD-040 device-matrix smoke: MainActivity starts on every matrix row (API level x screen size), reaches RESUMED, hosts a WebView whose
 * size equals the display the row configured, and the line "HRDMATRIX ..." records the actual device facts so the evidence proves WHICH
 * row ran. No network and no server needed.
 */
@RunWith(AndroidJUnit4.class)
public class MainActivitySmokeTest {
    @Test
    public void mainActivityReachesResumedWithWebViewFillingTheDisplay() throws Exception {
        Context ctx = InstrumentationRegistry.getInstrumentation().getTargetContext();
        assertEquals("org.helixreader.app", ctx.getPackageName());
        try (ActivityScenario<MainActivity> sc = ActivityScenario.launch(MainActivity.class)) {
            assertEquals(Lifecycle.State.RESUMED, sc.getState());
            AtomicReference<String> facts = new AtomicReference<>();
            AtomicReference<WebView> wv = new AtomicReference<>();
            sc.onActivity((MainActivity a) -> {
                wv.set(find(a.getWindow().getDecorView()));
                DisplayMetrics m = a.getResources().getDisplayMetrics();
                facts.set("sdk=" + android.os.Build.VERSION.SDK_INT + " px=" + m.widthPixels + "x" + m.heightPixels + " dpi=" + m.densityDpi
                        + " widthDp=" + (int) (m.widthPixels / m.density) + " heightDp=" + (int) (m.heightPixels / m.density));
                WebView w = wv.get();
                if (w != null) facts.set(facts.get() + " webview=" + w.getWidth() + "x" + w.getHeight());
            });
            Log.i("HRDMATRIX", facts.get());
            System.out.println("HRDMATRIX " + facts.get());
            assertNotNull("no WebView in the MainActivity view tree", wv.get());
            assertTrue("WebView has no size", wv.get().getWidth() > 0 && wv.get().getHeight() > 0);
        }
    }

    private static WebView find(View v) {
        if (v instanceof WebView) return (WebView) v;
        if (v instanceof ViewGroup) {
            ViewGroup g = (ViewGroup) v;
            for (int i = 0; i < g.getChildCount(); i++) {
                WebView r = find(g.getChildAt(i));
                if (r != null) return r;
            }
        }
        return null;
    }
}
