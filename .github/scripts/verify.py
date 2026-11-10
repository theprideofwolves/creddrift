"""Free, local verification; scans are confined to generated synthetic fixtures."""
import datetime,json,pathlib,re,subprocess,sys,tempfile
ROOT=pathlib.Path(__file__).resolve().parents[2]
def run(args):
    p=subprocess.run(args,cwd=ROOT,capture_output=True,text=True,encoding='utf-8',errors='replace')
    if p.returncode:
        print('FAILED: '+ ' '.join(args),file=sys.stderr)
        # Do not publish unsanitized test output or private launcher addresses.
        raise SystemExit(p.returncode)
    return p.stdout
revision=run(['git','rev-parse','HEAD']).strip()
with tempfile.TemporaryDirectory(prefix='creddrift-check-') as tmp:
    coverage=str(pathlib.Path(tmp)/'coverage.out')
    print('Running race-enabled functional tests...',flush=True)
    lines=run(['go','test','-race','-count=1','-json','-coverprofile='+coverage,'./...'])
    events=[json.loads(line) for line in lines.splitlines() if line.startswith('{')]
    tests=[{'package':e['Package'],'test':e['Test'],'outcome':e['Action'],'seconds':e.get('Elapsed',0)} for e in events if e.get('Test') and e.get('Action') in ('pass','fail','skip')]
    packages=[{'package':e['Package'],'outcome':e['Action']} for e in events if not e.get('Test') and e.get('Action') in ('pass','fail','skip')]
    cov=run(['go','tool','cover','-func='+coverage]);total=float(re.search(r'total:.*?([\d.]+)%',cov).group(1))
    print('Running vet, root build and packaged smoke...',flush=True)
    run(['go','vet','./...'])
    exe=str(pathlib.Path(tmp)/('creddrift.exe' if sys.platform=='win32' else 'creddrift'))
    run(['go','build','-trimpath','-o',exe,'.'])
    run([sys.executable,'.github/scripts/smoke.py',exe])
    print('Measuring synthetic detection and scanner-only performance...',flush=True)
    run(['go','run','./cmd/evaluate','-revision',revision,'-out','results/evaluation.json'])
    top=[t for t in tests if '/' not in t['test']]
    result={'revision':revision,'recorded_at_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'go':run(['go','version']).strip(),'gates':{'race_tests':'pass','vet':'pass','root_build':'pass','packaged_smoke':'pass','synthetic_evaluation':'pass'},'statement_coverage_percent':total,'top_level_passed':sum(t['outcome']=='pass' for t in top),'top_level_skipped':sum(t['outcome']=='skip' for t in top),'packages':packages,'tests':tests,'manual_checks':'not performed; see TESTING.md','live_integrations':'not performed; Jira tests are local mocks'}
    (ROOT/'results/verification.json').write_text(json.dumps(result,indent=2)+'\n',encoding='utf-8')
    print(f"PASS: {result['top_level_passed']} top-level tests; {result['top_level_skipped']} skips; {total}% statement coverage")
