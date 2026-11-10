import html,json,os,re,subprocess,sys,tempfile,threading,queue,urllib.request,urllib.parse,http.cookiejar
from pathlib import Path
executable=sys.argv[1]
urllib.request.install_opener(urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar())))
def launch(env):
 p=subprocess.Popen([str(Path(executable).resolve())],env=env,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True,encoding='utf-8');q=queue.Queue();threading.Thread(target=lambda:q.put(p.stdout.readline()),daemon=True).start()
 try:line=q.get(timeout=30)
 except queue.Empty:p.kill();p.wait();raise RuntimeError('no launcher URL')
 match=re.search(r'http://\S+',line)
 if not match:p.kill();p.wait();raise RuntimeError('launcher failed: '+line)
 return p,match.group(0)
def get(url):return urllib.request.urlopen(url,timeout=30).read().decode()
def post(url,fields):
 data=urllib.parse.urlencode(fields).encode()
 return urllib.request.urlopen(urllib.request.Request(url,data=data,headers={'Content-Type':'application/x-www-form-urlencoded'}),timeout=30).read().decode()
with tempfile.TemporaryDirectory(prefix='creddrift-',dir=os.environ.get('CREDDRIFT_SMOKE_TEMP')) as root:
 env=dict(os.environ,CREDDRIFT_DATA_DIR=root,CREDDRIFT_NO_BROWSER='1')
 p,url=launch(env)
 try:
  page=get(url);assert 'Try the guided demo' in page and 'Use my data' in page and 'ZgotmplZ' not in page
  token=urllib.parse.parse_qs(urllib.parse.urlparse(url).query)['token'][0]
  def route(path):return urllib.parse.urljoin(url,path)+'?'+urllib.parse.urlencode({'token':token})
  assert 'data-theme="light"' in page
  dark=post(route('/theme'),{'theme':'dark','return':'/setup'})
  assert 'data-theme="dark"' in dark and 'Light mode' in dark
  assert 'data-theme="dark"' in get(url), 'theme lost on navigation'
  light=post(route('/theme'),{'theme':'light','return':'/welcome'})
  assert 'data-theme="light"' in light and 'Dark mode' in light
  setup=get(route('/setup'));assert 'Set up your first task' in setup
  assert not list(Path(root).glob('*/onboarding.json')), 'viewing setup marked guide complete'
  assert 'Choose folder...' in setup and 'Folder to review' in setup
  page=post(url+'&onboarding=demo',{'action':'demo'})
  assert 'id="results"' in page and 'href="#results"' in page
  assert 'Your first result is ready' in page and 'Guided demo - synthetic data' in page and 'ZgotmplZ' not in page
  assert 'Potential secret' in page and 'synthetic-demo-only-abc123' not in page
  link=re.search(r'href="(/export[^\"]+)"',page);assert link,'missing export'
  payload=get(urllib.parse.urljoin(url,html.unescape(link.group(1))));value=json.loads(payload)
  assert len(value)==1 and 'synthetic-demo-only-abc123' not in payload
  assert value[0]['BlastRadius']==1 and value[0]['LastRotated']=='Not recorded'
  timeline=get(route('/timeline'));assert 'Evidence timeline' in timeline and 'new' in timeline
  evidence=json.loads(get(route('/export')+'&view=timeline'))
  assert len(evidence)==1 and evidence[0]['Outcome']=='complete'
  assert evidence[0]['Events'][0]['State']=='new'
  assert 'synthetic-demo-only-abc123' not in json.dumps(evidence)
  blocked=post(url,{'action':'ticket','hash':value[0]['Hash']})
  assert 'Synthetic findings cannot create real Jira tickets' in blocked
  assert list(Path(root).glob('*/onboarding.json')),'completion not remembered'
  assert 'Try the guided demo' in get(route('/welcome'))
  assert json.loads(get(urllib.parse.urljoin(url,html.unescape(link.group(1)))))==value,'restart guide modified results'
  try:get(url.split('?')[0]);raise AssertionError('missing token allowed')
  except urllib.error.HTTPError as e:assert e.code==403
 finally:p.terminate();p.wait(timeout=15)
 for lock in Path(root).glob('*/application.lock'):lock.unlink()
 p,url=launch(env)
 try:
  page=get(url);assert 'Try the guided demo' not in page and 'Welcome back' in page
  assert 'Potential secret' in page
  token=urllib.parse.parse_qs(urllib.parse.urlparse(url).query)['token'][0]
  source=Path(root,'source').resolve();source.mkdir()
  candidate='synthetic-own-folder-only-5678'
  file=source/'app.env';file.write_text('password='+candidate+'\n',encoding='utf-8')
  setup=post(route('/setup/check'),{'check':'input','path':str(source)})
  assert 'Folder is readable' in setup and candidate not in setup
  assert len(json.loads(get(route('/export'))))==1,'setup ran a scan'
  page=post(url,{'action':'scan','path':str(source)})
  assert 'Complete:' in page and candidate not in page
  current=json.loads(get(route('/export')))
  own=[row for row in current if row['FilePath']==str(file)]
  assert len(current)==2 and len(own)==1 and own[0]['BlastRadius']==1
  post(url,{'action':'ignore','hash':own[0]['Hash']})
  assert len(json.loads(get(route('/export'))))==1
  assert len(json.loads(get(route('/export')+'&view=ignored')))==1
  post(url,{'action':'restore','hash':own[0]['Hash']})
  file.write_text('x'*(2<<20),encoding='utf-8')
  page=post(url,{'action':'scan','path':str(source)})
  assert 'Scan incomplete; existing findings retained' in page
  history=json.loads(get(route('/export')+'&view=timeline'))
  assert history[0]['Outcome']=='failed' and history[0]['Events']==[]
  assert len(json.loads(get(route('/export'))))==2
  file.unlink()
  post(url,{'action':'scan','path':str(source)})
  assert len(json.loads(get(route('/export'))))==1
  retired=json.loads(get(route('/export')+'&view=retired'))
  assert len(retired)==1 and retired[0]['Hash']==own[0]['Hash'] and retired[0]['BlastRadius']==0
  history=json.loads(get(route('/export')+'&view=timeline&hash='+own[0]['Hash']))
  assert history[0]['Events'][0]['State']=='absent from checked coverage'
  assert all(e['Hash']==own[0]['Hash'] for scan in history for e in scan['Events'])
 finally:p.terminate();p.wait(timeout=15)
 print('CredDrift: packaged onboarding, demo isolation, redaction, persistence, folder scan, review, failed rescan and retired export PASS')
