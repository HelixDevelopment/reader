import http.server,sys,urllib.request,json
UP="http://127.0.0.1:18417"; MODE=sys.argv[2]; LOG=sys.argv[3]
OP=urllib.request.build_opener(urllib.request.ProxyHandler({}))
n={"c":0}
class H(http.server.BaseHTTPRequestHandler):
    def do_POST(s):
        raw=s.rfile.read(int(s.headers.get("Content-Length") or 0)); b=json.loads(raw or b"{}")
        act=b.get("action"); obj=b.get("object")
        if act=="create" and obj=="ticket":
            n["c"]+=1
            if n["c"]==3:
                open(LOG,"a").write("INJECT-GARBAGE-200-HTML create#3\n")
                body=b"<html><body>502 Bad Gateway from corp proxy</body></html>"
                s.send_response(200); s.send_header("Content-Type","text/html"); s.send_header("Content-Length",str(len(body))); s.end_headers(); s.wfile.write(body); return
        r=urllib.request.Request(UP+s.path,raw,{k:v for k,v in s.headers.items() if k.lower() in("content-type","authorization")},method="POST")
        try:
            with OP.open(r,timeout=60) as x: c,o=x.status,x.read()
        except urllib.error.HTTPError as e: c,o=e.code,e.read()
        open(LOG,"a").write("FWD %s %s %d\n"%(act,obj,c))
        s.send_response(c); s.send_header("Content-Type","application/json"); s.send_header("Content-Length",str(len(o))); s.end_headers(); s.wfile.write(o)
    def log_message(s,*a): pass
http.server.ThreadingHTTPServer(("127.0.0.1",int(sys.argv[1])),H).serve_forever()
