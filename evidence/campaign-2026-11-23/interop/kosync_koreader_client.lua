-- HRD-026 probe: drives KOReader's REAL kosync plugin client (KOSyncClient.lua + Spore + api.json,
-- shipped in the installed koreader deb) against a Helix server. Only ui/uimanager, logger and
-- socketutil are stubbed (they are GUI/log glue, not protocol code). Run with koreader's bundled luajit
-- from /usr/lib/koreader:  ./luajit <this file> http://127.0.0.1:PORT [user] [epub-path]
package.path = "common/?.lua;frontend/?.lua;plugins/kosync.koplugin/?.lua;" .. package.path
package.cpath = "common/?.so;/usr/lib/lua/?.so;" .. package.cpath
local base = arg[1]
package.preload["ui/uimanager"] = function() return {looper = nil} end
package.preload["logger"] = function() return {dbg=function(...) print("[dbg]", ...) end, warn=function(...) print("[warn]", ...) end, info=function() end, err=function(...) print("[err]", ...) end} end
package.preload["socketutil"] = function() return {set_timeout=function() end, reset_timeout=function() end} end
local f = io.open("plugins/kosync.koplugin/api.json"); local spec = f:read("*a"); f:close()
local json = require("json")
local KOSyncClient = dofile("plugins/kosync.koplugin/KOSyncClient.lua")
local c = KOSyncClient:new{service_spec = "plugins/kosync.koplugin/api.json", custom_url = base}
local md5 = require("ffi/sha2").md5
local user, key = (arg[2] or "koreader-probe"), md5("probe-password")
local function show(label, ok, res, body) print(string.format("RESULT %-28s ok=%s status=%s body=%s", label, tostring(ok), tostring(res), type(body)=="table" and json.encode(body) or tostring(body))) end
local ok, b = c:register(user, key); show("register", ok, ok, b)
local ok2, b2 = c:authorize(user, key); show("authorize(valid)", ok2, ok2, b2)
local ok3, b3 = c:authorize(user, md5("wrong")); show("authorize(wrong key)", ok3, ok3, b3)
-- Document id: KOReader'"'"'s REAL util.partialMD5 (frontend/util.lua) over a real EPUB (arg[3]) when given.
local doc = "b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0"
if arg[3] then
    -- ffi/utf8proc needs a native lib unrelated to hashing; stub only that import so the REAL util.lua loads.
    package.preload["ffi/utf8proc"] = function() return {} end
    local util = require("util")
    doc = assert(util.partialMD5(arg[3]), "partialMD5 failed")
    print("RESULT partialMD5(" .. arg[3] .. ") = " .. doc)
end
local prog = "/body/DocFragment[3]/body/p[12]/text().5"
c:update_progress(user, key, doc, nil, prog, 0.4217, "Helix Interop Probe", "dev-hrd026", function(ok, st, body) show("update_progress", ok, st, body) end)
c:get_progress(user, key, doc, function(ok, body) show("get_progress", ok, ok, body) end)
c:get_progress(user, key, "00000000000000000000000000000000", function(ok, body) show("get_progress(unknown doc)", ok, ok, body) end)
-- second registration of the same user: reference server answers 402 (api.json register expects [201, 402]).
local ok4, b4 = c:register(user, key); show("register(duplicate)", ok4, ok4, b4)
