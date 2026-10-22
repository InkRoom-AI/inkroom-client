import {test} from 'node:test';
import assert from 'node:assert/strict';
import {execFile} from 'node:child_process';
import {promisify} from 'node:util';
import {fileURLToPath} from 'node:url';
import {PromptClient, serviceOrigin} from '../src/client.mjs';

const run = promisify(execFile);
const script = fileURLToPath(new URL('../skills/inkroom-prompts/scripts/prompts.mjs',import.meta.url));

test('malformed origin configuration fails with a bounded public error',()=>{
  for (const origin of [null,{},'', 'not an origin',' https://prompts.inkroom.ai','https://prompts.inkroom.ai\n','https://user:secret@prompts.inkroom.ai','https://prompts.inkroom.ai/path','https://prompts.inkroom.ai?key=secret']) {
    assert.throws(()=>serviceOrigin(origin),{code:'invalid_origin'});
  }
  assert.equal(serviceOrigin('https://operator.example.test'),'https://operator.example.test');
  assert.equal(serviceOrigin('http://127.0.0.1:8788'),'http://127.0.0.1:8788');
});

test('prompt client rejects destination, path and method changes before any fetch',async()=>{
  let calls=0;
  const client=new PromptClient({fetchImpl:async()=>{calls++;throw Error('must not contact network');}});
  for (const [path,options] of [
    ['@example.test/api/prompts/assistant/search',{}],
    ['//example.test/api/prompts/assistant/search',{}],
    ['https://example.test/api/prompts/assistant/search',{}],
    ['/\\example.test/api/prompts/assistant/search',{}],
    ['/api/prompts/assistant/search\n',{}],
    ['/api/prompts/assistant/search#fragment',{}],
    ['/api/prompts/assistant/search',{body:{query:'poster'}}],
    ['/api/prompts/assistant/optimize',{}],
    ['/api/prompts/assistant/optimize?extra=1',{body:{query:'poster'}}],
    ['/api/images/generations',{body:{prompt:'poster'}}],
    ['/api/../../api/images/generations',{body:{prompt:'poster'}}],
    [null,{}],
  ]) await assert.rejects(client.request(path,options),{code:'invalid_request'});
  assert.equal(calls,0);
});

test('shell-shaped multilingual input remains one JSON/query value without execution',async()=>{
  const query='海报 "quoted" $(touch never-run) `echo never-run`; & 照明';
  const requests=[];
  const client=new PromptClient({fetchImpl:async(url,options)=>{
    requests.push({url,options});
    return new Response(JSON.stringify(options.method==='POST'?{kind:'optimized',prompt:query}:{kind:'library',items:[]}),{headers:{'content-type':'Application/JSON; charset=utf-8'}});
  }});
  assert.deepEqual((await client.search(query)).items,[]);
  assert.equal((await client.optimize(query)).prompt,query);
  assert.equal(new URL(requests[0].url).searchParams.get('query'),query);
  assert.deepEqual(JSON.parse(requests[1].options.body),{query});
  for (const request of requests) {
    assert.equal(request.options.credentials,'omit');
    assert.equal(request.options.redirect,'error');
    assert.equal(request.options.headers.Authorization,undefined);
  }
});

test('standalone CLI rejects extra arguments and unknown actions without network',async()=>{
  const preload='data:text/javascript,'+encodeURIComponent("globalThis.fetch=async()=>{throw new Error('UNEXPECTED_NETWORK_REQUEST')};");
  for (const args of [['search','poster','extra'],['optimize','poster','extra'],['models','poster'],['capabilities','extra'],['--eval','process.exit(0)'],['search','']]) {
    try {
      await run(process.execPath,['--import',preload,script,...args],{env:{PATH:process.env.PATH || ''}});
      assert.fail('Invalid arguments must fail');
    } catch (error) {
      assert.equal(error.code,2);
      assert.match(error.stderr,/Usage:/);
      assert.doesNotMatch(error.stdout+error.stderr,/UNEXPECTED_NETWORK_REQUEST/);
    }
  }
});
