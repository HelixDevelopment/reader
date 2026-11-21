import json, re, shutil, subprocess, sys, time
ROOT='/home/milosvasic/Projects/helix_reader'
SRV=ROOT+'/server'
DSN='postgres://helixqual@127.0.0.1:34320/helix_test?sslmode=disable'
import os
ENV=dict(os.environ, HELIX_TEST_POSTGRES_DSN=DSN)

NEEDLES=[
 dict(id='N1-catalog-validate-noop', pkg='./internal/catalog/', tests='TestPoolSettingsValidation|TestNewPostgresRejectsPoolBeforeDialing',
      path='internal/catalog/postgres.go', old='func (s poolSettings) validate() error {\n',
      new='func (s poolSettings) validate() error {\n\tif true {\n\t\treturn nil\n\t}\n'),
 dict(id='N2-catalog-drop-setmaxopen', pkg='./internal/catalog/', tests='TestWithPoolAppliesLimits',
      path='internal/catalog/postgres.go', old='\tdb.SetMaxOpenConns(settings.maxOpenConns)\n', new=''),
 dict(id='N3-aimemory-validate-noop', pkg='./internal/aimemory/', tests='TestPoolSettingsValidation|TestNewPostgresRejectsPoolBeforeDialing',
      path='internal/aimemory/postgres.go', old='func (s poolSettings) validate() error {\n',
      new='func (s poolSettings) validate() error {\n\tif true {\n\t\treturn nil\n\t}\n'),
 dict(id='N4-aimemory-drop-setmaxopen', pkg='./internal/aimemory/', tests='TestWithPoolAppliesLimits',
      path='internal/aimemory/postgres.go', old='\tdb.SetMaxOpenConns(settings.maxOpenConns)\n', new=''),
 dict(id='N5-config-applyPoolEnv-noop', pkg='./internal/config/', tests='TestPoolConfiguration',
      path='internal/config/config.go', old='func applyPoolEnv(prefix string, target *Pool) error {\n',
      new='func applyPoolEnv(prefix string, target *Pool) error {\n\tif true {\n\t\treturn nil\n\t}\n'),
 dict(id='N6-config-drop-idle-crosscheck', pkg='./internal/config/', tests='TestPoolConfiguration/idle_above_open_rejected',
      path='internal/config/config.go', old='\tif target.MaxIdleConns > target.MaxOpenConns {\n',
      new='\tif false && target.MaxIdleConns > target.MaxOpenConns {\n'),
]

results=[]
for n in NEEDLES:
    src=os.path.join(SRV,n['path']); bak=src+'.needlebak'
    shutil.copy2(src,bak)
    body=open(src,encoding='utf-8').read()
    assert n['old'] in body, f"{n['id']}: needle target absent — the code moved, not a pass"
    open(src,'w',encoding='utf-8').write(body.replace(n['old'],n['new'],1))
    p=subprocess.run(['go','test',n['pkg'],'-run',n['tests'],'-count=1','-v'],cwd=SRV,env=ENV,capture_output=True,text=True)
    failed = p.returncode!=0
    killed=[l.strip() for l in p.stdout.splitlines() if l.startswith('--- FAIL') or l.startswith('    --- FAIL')]
    shutil.move(bak,src)
    verify=subprocess.run(['go','test',n['pkg'],'-run',n['tests'],'-count=1'],cwd=SRV,env=ENV,capture_output=True,text=True)
    results.append(dict(id=n['id'],needle='killed' if failed else 'SURVIVED',
                        fail_lines=killed[:4], restored_pass=verify.returncode==0,
                        restored_tail=verify.stdout.strip().splitlines()[-1] if verify.stdout.strip() else verify.stderr.strip()[-200:]))
    print(json.dumps(results[-1]),flush=True)
bad=[r for r in results if r['needle']!='killed' or not r['restored_pass']]
out=dict(generated_at=time.strftime('%Y-%m-%dT%H:%M:%S%z'),dsn_host='127.0.0.1:34320 (helixqual-pg, in-host client)',
         verdict='PASS' if not bad else 'FAIL', needles=results,
         note='Each needle deletes the behavior under test. SURVIVED means the test has no detection power. restored_pass confirms the mutation was reverted and the suite is green again.')
open(ROOT+'/evidence/campaign-2026-11-20/pool-needles/needle-results.json','w').write(json.dumps(out,indent=2)+'\n')
print('VERDICT','PASS' if not bad else 'FAIL')
sys.exit(1 if bad else 0)
