-- HRD-026: second-device pull using KOReader's REAL kosync client + REAL util.partialMD5. Usage (cwd /usr/lib/koreader):
-- ./luajit kosync_pull_probe.lua http://127.0.0.1:PORT user password epub-path device_id
package.path = "common/?.lua;frontend/?.lua;plugins/kosync.koplugin/?.lua;" .. package.path
package.cpath = "common/?.so;/usr/lib/lua/?.so;" .. package.cpath
package.preload["ui/uimanager"] = function() return {looper = nil} end
package.preload["logger"] = function() return {dbg=function() end, warn=function() end, info=function() end, err=function(...) print("[err]", ...) end} end
package.preload["socketutil"] = function() return {set_timeout=function() end, reset_timeout=function() end} end
package.preload["ffi/utf8proc"] = function() return {} end
local json = require("json")
local KOSyncClient = dofile("plugins/kosync.koplugin/KOSyncClient.lua")
local c = KOSyncClient:new{service_spec = "plugins/kosync.koplugin/api.json", custom_url = arg[1]}
local md5 = require("ffi/sha2").md5
local util = require("util")
local doc = assert(util.partialMD5(arg[4]))
c:get_progress(arg[2], md5(arg[3]), doc, function(ok, body)
  print(string.format("RESULT pull device=%s doc=%s ok=%s body=%s", arg[5], doc, tostring(ok), json.encode(body))) end)
