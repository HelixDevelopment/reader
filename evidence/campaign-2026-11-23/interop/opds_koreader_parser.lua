-- HRD-026 probe: feeds Helix OPDS 1.2 feed bytes to KOReader's REAL opdsparser.lua (installed koreader deb).
-- Usage (cwd /usr/lib/koreader): ./luajit <this> <feed.xml>
package.path = "common/?.lua;frontend/?.lua;plugins/opds.koplugin/?.lua;" .. package.path
package.cpath = "common/?.so;/usr/lib/lua/?.so;" .. package.cpath
require("ffi/loadlib")
local f = assert(io.open(arg[1], "rb")); local xml = f:read("*a"); f:close()
local P = dofile("plugins/opds.koplugin/opdsparser.lua")
local feed = P:parse(xml); if feed.feed then feed = feed.feed end
print("RESULT feed.title=", feed.title, "entries=", feed.entry and #feed.entry or "single")
local entries = feed.entry or {}
if entries.title then entries = {entries} end
for i, e in ipairs(entries) do
  print("ENTRY", i, e.title)
  local links = e.link or {}
  if links.href then links = {links} end
  for _, l in ipairs(links) do
    local cnt
    for k, v in pairs(l) do if type(k) == "string" and k:sub(-6) == ":count" then cnt = tonumber(v) end end
    print("   LINK rel=" .. tostring(l.rel) .. " type=" .. tostring(l.type) .. " href=" .. tostring(l.href) .. " stream_count(opdsbrowser.lua:826 rule)=" .. tostring(cnt))
  end
end
local fl = feed.link or {}
if fl.href then fl = {fl} end
for _, l in ipairs(fl) do print("FEEDLINK rel=" .. tostring(l.rel) .. " href=" .. tostring(l.href)) end
