import {test} from 'node:test';
import assert from 'node:assert/strict';
import {createServer as createHTTPServer} from 'node:http';
import {spawn} from 'node:child_process';
import {mkdtempSync,rmSync,writeFileSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {dirname,join} from 'node:path';
import {fileURLToPath,pathToFileURL} from 'node:url';
import {Client} from '@modelcontextprotocol/sdk/client/index.js';
import {StdioClientTransport} from '@modelcontextprotocol/sdk/client/stdio.js';
import {StreamableHTTPClientTransport} from '@modelcontextprotocol/sdk/client/streamableHttp.js';
import {McpServer} from '@modelcontextprotocol/sdk/server/mcp.js';
import {StreamableHTTPServerTransport} from '@modelcontextprotocol/sdk/server/streamableHttp.js';
import {z} from 'zod';
import {PublicCatalogClient,publicCapabilities} from '../src/client.mjs';
import {createHttpApp} from '../src/http.mjs';

const cwd=dirname(dirname(fileURLToPath(import.meta.url)));
const sale={scope:'public_catalog',accountAuthorization:'not_evaluated',pricingSource:{kind:'inkroom_public_sale',url:'https://inkroom.ai/image/models',unit:'credits',usdPerCredit:0.01,priceSemantics:'public_sale_exact'},models:[{id:'fixture-public-model',status:'listed',offers:[{pricing:{free:false,basePoints:0.125,status:'configured',unit:'credits'}}]}],visibleCount:1};
async function catalogFixture({sse=false,status=200,toolError=false,empty=false}={}) {
 const requests=[];let toolCalls=0;
 const http=createHTTPServer(async(req,res)=>{
  let raw='';for await(const part of req)raw+=part;
  const body=raw?JSON.parse(raw):undefined;
  requests.push({method:req.method,path:req.url,headers:req.headers,body});
  if(status!==200){res.writeHead(status,{'content-type':'application/json','retry-after':'37'});res.end('{}');return;}
  if(req.method!=='POST'){res.writeHead(405);res.end();return;}
  const server=new McpServer({name:'public-catalog-fixture',version:'1'});
  server.registerTool('list_models',{inputSchema:{includeUnavailable:z.boolean().optional()}},async(args)=>{
   toolCalls++;assert.equal(args.includeUnavailable,false);
   if(toolError)return{isError:true,content:[{type:'text',text:'public config unavailable'}]};
   const value=empty?{...sale,models:[],visibleCount:0}:sale;
   return{content:[{type:'text',text:JSON.stringify(value)}],structuredContent:value};
  });
  const transport=new StreamableHTTPServerTransport({sessionIdGenerator:undefined,enableJsonResponse:!sse});
  res.on('close',()=>{void transport.close();void server.close();});
  await server.connect(transport);await transport.handleRequest(req,res,body);
 });
 await new Promise(resolve=>http.listen(0,'127.0.0.1',resolve));
 return{url:`http://127.0.0.1:${http.address().port}/mcp`,requests,toolCalls:()=>toolCalls,close:async()=>{http.closeAllConnections();await new Promise(resolve=>http.close(resolve));}};
}
function assertPublicRequests(fixture){
 assert.ok(fixture.requests.every(row=>row.headers.authorization===undefined&&row.headers.cookie===undefined));
 assert.ok(fixture.requests.every(row=>row.method==='POST'&&row.path==='/mcp'));
 const calls=fixture.requests.filter(row=>row.body?.method==='tools/call');
 assert.ok(calls.every(row=>row.body.params.name==='list_models'));
}

test('native public catalog client interoperates with real SDK JSON and SSE, preserves prices and uses no credentials',async()=>{
 for(const sse of [false,true]){
  const fixture=await catalogFixture({sse});
  try{
   const client=new PublicCatalogClient({endpoint:fixture.url});
   assert.deepEqual(await client.models(),sale);
   assert.equal(fixture.toolCalls(),1);
   assert.deepEqual(fixture.requests.map(row=>row.body.method),['initialize','notifications/initialized','tools/call']);
   assertPublicRequests(fixture);
  }finally{await fixture.close();}
 }
});

test('real public empty catalog stays empty; tool failure, 401 and 429 are errors without optimization or retry',async()=>{
 for(const options of [{empty:true},{toolError:true},{status:401},{status:429}]){
  const fixture=await catalogFixture(options);
  try{
   const client=new PublicCatalogClient({endpoint:fixture.url});
   if(options.empty)assert.equal((await client.models()).models.length,0);
   else await assert.rejects(client.models(),error=>options.status===429?error.code==='rate_limited'&&error.retryAfter===37:typeof error.code==='string');
   assert.equal(fixture.requests.filter(row=>row.body?.method==='initialize').length,1);
   assert.ok(fixture.toolCalls()<=1);assertPublicRequests(fixture);
  }finally{await fixture.close();}
 }
});

test('catalog refuses credential URLs, malformed payloads, oversized responses, cancellation and offline without retry',async()=>{
 for(const endpoint of ['https://key:secret@mcp.inkroom.ai','http://example.test','https://mcp.inkroom.ai?token=x'])assert.throws(()=>new PublicCatalogClient({endpoint}));
 let calls=0;
 const offline=new PublicCatalogClient({fetchImpl:async()=>{calls++;throw new TypeError('offline');}});
 await assert.rejects(offline.models(),{code:'network_error'});assert.equal(calls,1);
 const cancelled=new AbortController();cancelled.abort();
 await assert.rejects(new PublicCatalogClient({fetchImpl:async(_url,options)=>{options.signal.throwIfAborted();}}).models({signal:cancelled.signal}),{code:'timeout_or_cancelled'});
 for(const text of ['not json','x'.repeat(512*1024+1)]){
  const client=new PublicCatalogClient({fetchImpl:async()=>new Response(text,{headers:{'content-type':'application/json'}})});
  await assert.rejects(client.models(),{code:text==='not json'?'invalid_response':'response_too_large'});
 }
 assert.equal(publicCapabilities().serviceStatus,'not_checked');
});

test('real candidate HTTP advertises help first and bridges JSON/SSE catalog without prompt or optimizer callbacks',async()=>{
 const upstream=await catalogFixture({sse:true});let search=0,optimize=0;
 const listener=createHttpApp({search:async()=>{search++;throw Error('unexpected');},optimize:async()=>{optimize++;throw Error('unexpected');}},new PublicCatalogClient({endpoint:upstream.url})).listen(0,'127.0.0.1');
 await new Promise(resolve=>listener.once('listening',resolve));
 const client=new Client({name:'public-candidate-http',version:'1'});
 try{
  await client.connect(new StreamableHTTPClientTransport(new URL(`http://127.0.0.1:${listener.address().port}/mcp`)));
  const tools=await client.listTools();assert.equal(tools.tools[0].name,'help');assert.equal(tools.tools[0].annotations.readOnlyHint,true);assert.equal(tools.tools[0].annotations.openWorldHint,false);
  assert.deepEqual((await client.callTool({name:'help',arguments:{}})).structuredContent,publicCapabilities());assert.equal(upstream.requests.length,0);
  assert.deepEqual((await client.callTool({name:'list_models',arguments:{}})).structuredContent,sale);
  assert.equal(search,0);assert.equal(optimize,0);assertPublicRequests(upstream);
 }finally{await client.close();listener.closeAllConnections();await new Promise(resolve=>listener.close(resolve));await upstream.close();}
});

async function stdioCandidate(endpoint,{offline=false}={}){
 const imports={server:pathToFileURL(join(cwd,'src/server.mjs')).href,client:pathToFileURL(join(cwd,'src/client.mjs')).href};
 const code=`import {createPromptServer} from ${JSON.stringify(imports.server)};import {PublicCatalogClient} from ${JSON.stringify(imports.client)};import {StdioServerTransport} from '@modelcontextprotocol/sdk/server/stdio.js';const unexpected=async()=>{throw new Error('Unexpected prompt operation')};await createPromptServer({search:unexpected,optimize:unexpected},new PublicCatalogClient({endpoint:${JSON.stringify(endpoint)}${offline?",fetchImpl:async()=>{throw new TypeError('offline')}":''}})).connect(new StdioServerTransport());`;
 const client=new Client({name:'public-candidate-stdio',version:'1'});
 await client.connect(new StdioClientTransport({command:process.execPath,args:['--input-type=module','--eval',code],cwd,env:{INKROOM_MCP_TOKEN:'fixture-token-must-not-send',OPENAI_API_KEY:'fixture-key-must-not-send'},stderr:'pipe'}));
 return client;
}

test('real candidate STDIO supports credential-free static probe and model call with inherited dummy credentials ignored',async()=>{
 const fixture=await catalogFixture();const client=await stdioCandidate(fixture.url);
 try{
  assert.deepEqual((await client.listTools()).tools.map(tool=>tool.name),['help','list_models','search_prompts','optimize_prompt']);
  const help=await client.callTool({name:'help',arguments:{}});assert.equal(help.structuredContent.serviceStatus,'not_checked');assert.equal(fixture.requests.length,0);
  const result=await client.callTool({name:'list_models',arguments:{}});assert.deepEqual(result.structuredContent,sale);assertPublicRequests(fixture);
 }finally{await client.close();await fixture.close();}
});

async function skillProcess(endpoint,action,{offline=false}={}){
 const temp=mkdtempSync(join(tmpdir(),'inkroom-public-skill-'));
 const preload=join(temp,'route-fetch.mjs');
 writeFileSync(preload,`const original=globalThis.fetch;globalThis.fetch=(url,options)=>{if(String(url)!=='https://mcp.inkroom.ai/')throw new Error('Unexpected endpoint');${offline?"throw new TypeError('offline');":`return original(${JSON.stringify(endpoint)},options);`}};`);
 try{
  return await new Promise((resolve,reject)=>{
   const child=spawn(process.execPath,['--import',pathToFileURL(preload).href,'skills/inkroom-prompts/scripts/prompts.mjs',action],{cwd,env:{HOME:temp,USERPROFILE:temp,INKROOM_MCP_TOKEN:'fixture-token-must-not-send',OPENAI_API_KEY:'fixture-key-must-not-send'},stdio:['ignore','pipe','pipe']});
   let stdout='',stderr='';child.stdout.on('data',part=>{stdout+=part;});child.stderr.on('data',part=>{stderr+=part;});child.on('error',reject);child.on('close',exitCode=>resolve({exitCode,stdout,stderr}));
  });
 }finally{rmSync(temp,{recursive:true,force:true});}
}

test('real standalone Skill capabilities is local and models preserves sale source over SDK SSE without credentials',async()=>{
 const fixture=await catalogFixture({sse:true});
 try{
  const local=await skillProcess(fixture.url,'capabilities');assert.equal(local.exitCode,0,local.stderr);assert.equal(JSON.parse(local.stdout).serviceStatus,'not_checked');assert.equal(fixture.requests.length,0);
  const models=await skillProcess(fixture.url,'models');assert.equal(models.exitCode,0,models.stderr);assert.deepEqual(JSON.parse(models.stdout),sale);assertPublicRequests(fixture);
 }finally{await fixture.close();}
});

test('real standalone Skill offline/401/429 and candidate tool failures remain explicit without optimization or identity fallback',async()=>{
 for(const status of [401,429,'offline']){
  const offline=status==='offline';const fixture=await catalogFixture({status:offline?200:status});
  try{
   const result=await skillProcess(fixture.url,'models',{offline});assert.equal(result.exitCode,1);const error=JSON.parse(result.stdout).error;assert.equal(error.code,offline?'network_error':status===401?'public_catalog_unavailable':'rate_limited');if(status===429)assert.equal(error.retryAfter,37);
   assert.equal(fixture.requests.length,offline?0:1);assert.equal(fixture.toolCalls(),0);assertPublicRequests(fixture);
   const client=await stdioCandidate(fixture.url,{offline});
   try{const call=await client.callTool({name:'list_models',arguments:{}});assert.equal(call.isError,true);assert.equal(JSON.parse(call.content[0].text).error.code,error.code);}finally{await client.close();}
   assert.equal(fixture.requests.length,offline?0:2);assertPublicRequests(fixture);
   let optimize=0;let search=0;
   const catalog=new PublicCatalogClient({endpoint:fixture.url,...(offline?{fetchImpl:async()=>{throw new TypeError('offline');}}:{})});
   const listener=createHttpApp({search:async()=>{search++;},optimize:async()=>{optimize++;}},catalog).listen(0,'127.0.0.1');await new Promise(resolve=>listener.once('listening',resolve));
   const http=new Client({name:'public-candidate-errors',version:'1'});
   try{
    await http.connect(new StreamableHTTPClientTransport(new URL(`http://127.0.0.1:${listener.address().port}/mcp`)));
    const call=await http.callTool({name:'list_models',arguments:{}});assert.equal(call.isError,true);assert.equal(JSON.parse(call.content[0].text).error.code,error.code);
    assert.equal(optimize,0);assert.equal(search,0);
   }finally{await http.close();listener.closeAllConnections();await new Promise(resolve=>listener.close(resolve));}
   assert.equal(fixture.requests.length,offline?0:3);assertPublicRequests(fixture);
  }finally{await fixture.close();}
 }
});
