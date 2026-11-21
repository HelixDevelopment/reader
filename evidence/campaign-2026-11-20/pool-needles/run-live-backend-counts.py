import json, subprocess, time, urllib.request, os
DSN='postgres://helixqual@127.0.0.1:34320/helix_test?sslmode=disable'
PORT=18099
URL=f'http://127.0.0.1:{PORT}/v1/works?q=Scale%20Volume%2042&limit=20'
Q="SELECT count(*) FROM pg_stat_activity WHERE usename='helixqual' AND client_addr IS NOT NULL"
def backends():
    r=subprocess.run(['podman','exec','helixqual-pg','psql','-U','helixqual','-d','helix_test','-Atc',Q],capture_output=True,text=True)
    try: return int(r.stdout.strip())
    except ValueError: return -1
def case(label, extra):
    env=dict(os.environ, HELIX_ADDRESS=f'127.0.0.1:{PORT}', HELIX_POSTGRES_DSN=DSN, **extra)
    log=open(f'/tmp/pool-{label}.log','w')
    srv=subprocess.Popen(['/tmp/helix-pool-server'],env=env,stdout=log,stderr=subprocess.STDOUT)
    for _ in range(80):
        try:
            urllib.request.urlopen(f'http://127.0.0.1:{PORT}/healthz',timeout=1); break
        except Exception: time.sleep(0.2)
    procs=[subprocess.Popen(['curl','-s','-o','/dev/null',URL]) for _ in range(12)]
    peak=0; samples=[]
    deadline=time.time()+12
    while time.time()<deadline:
        n=backends(); samples.append(n); peak=max(peak,n)
        if all(p.poll() is not None for p in procs) and max(samples[-3:] or [0])==0: break
    for p in procs: p.wait()
    srv.terminate(); srv.wait(timeout=20); log.close()
    return dict(label=label, env=extra, peak_backends=peak, samples=samples)
out=[case('capped',{'HELIX_CATALOG_POOL_MAX_OPEN_CONNS':'3','HELIX_CATALOG_POOL_MAX_IDLE_CONNS':'3'}), case('default',{})]
path='/home/milosvasic/Projects/helix_reader/evidence/campaign-2026-11-20/pool-needles/live-backend-counts.json'
json.dump(dict(generated_at=time.strftime('%Y-%m-%dT%H:%M:%S%z'),
    method='12 concurrent /v1/works?q=Scale%20Volume%2042 (unanchored ILIKE, ~770 ms each on the 50k fixture) against the real helix-server binary; pg_stat_activity sampled from inside helixqual-pg over TCP backends only, so the sampling psql (unix socket) is not counted.',
    verdict_note='peak_backends must equal the configured cap when capped and exceed it by default, proving HELIX_CATALOG_POOL_* reaches the live pool rather than only the parsed config.',
    cases=out), open(path,'w'), indent=2)
print(json.dumps([{k:v[k] for k in ('label','peak_backends')} for v in out]))
print('capped samples nonzero:', [s for s in out[0]['samples'] if s>0][:12])
print('default samples nonzero:', [s for s in out[1]['samples'] if s>0][:14])
