import { expect, test, type Page } from '@playwright/test'
import type { MonthlyReview, ReviewDocument, ReviewFacts } from '../src/api/reviews'
const time='2026-10-10T04:00:00Z'
function facts(month:string,title='完成行动'):ReviewFacts{return {month,timezone:'Asia/Shanghai',from:month+'-01',to:month+'-28',generated_at:time,in_progress:month==='2026-10',actions:[{key:'task:1',kind:'task',source_id:1,parent_id:1,title,date:month+'-02 10:00',detail:'',state:'任务完成',domain:'',milestone:false,url:'/projects?task=1',available:true}],learning:[],events:[{key:'event:1',kind:'event',source_id:1,parent_id:0,title:'补录的经历',date:month+'-03',detail:'真实发生日期',state:'生活经历',domain:'home',milestone:true,url:'/life?event=1',available:true}],goals:[],finance:[{ledger_id:1,name:'钱迹',currency:'CNY',available:true,summary:null}],assets:month==='2026-09'?[{available:true,snapshot:{id:1,snapshot_date:month+'-15',currency:'CNY',bank_minor:'0',platform_minor:null,investment_minor:null,other_minor:null,debt_minor:'0',asset_total_minor:'0',net_asset_minor:'0',complete:false,note:'部分范围',version:1,created_at:time,updated_at:time}}]:[],limits:['升级前旧完成日期无法可靠统计。']}}
async function mock(page:Page){
 const state={saved:new Map<string,MonthlyReview>(),snapshots:new Map<number,ReviewFacts>(),history:new Map<string,number[]>(),archived:new Map<number,MonthlyReview>(),next:1,title:'完成行动',failSave:false,delayMonth:'',removed:false,writes:0,confirmations:[] as boolean[]}
 const copy=<T>(x:T):T=>JSON.parse(JSON.stringify(x))
 await page.route('**/api/v1/**',async route=>{
  const req=route.request(),path=new URL(req.url()).pathname
  const reply=(data:unknown)=>route.fulfill({status:200,contentType:'application/json',body:JSON.stringify({data})})
  if(path.endsWith('/auth/me'))return reply({id:1,username:'test',role:'admin',status:'active'})
  if(path==='/api/v1/life/reviews')return reply([...state.saved.values()])
  const match=path.match(/^\/api\/v1\/life\/reviews\/(\d{4}-\d{2})(?:\/(preview|history)(?:\/(\d+))?)?$/)
  if(!match)return reply([])
  const month=match[1]!,kind=match[2],saved=state.saved.get(month)
  const document=(id:number):ReviewDocument=>{const f=copy(state.snapshots.get(id)!);if(state.removed)f.actions[0]!.available=false;return {review:state.saved.get(month)??null,facts_id:id,facts:f,history:(state.history.get(month)??[]).map(id=>({id,generated_at:time,confirmed_at:time})),archived_personal:state.archived.get(id)}}
  if(kind==='history')return reply(document(Number(match[3])))
  if(kind==='preview'){if(state.delayMonth===month)await new Promise(resolve=>setTimeout(resolve,500));const id=state.next++;state.snapshots.set(id,facts(month,state.title));return reply({review:null,facts_id:id,facts:copy(state.snapshots.get(id))})}
  if(req.method()==='GET')return reply(saved?document(saved.facts_id):{review:null,facts_id:0})
  state.writes++;if(state.failSave)return route.fulfill({status:500,contentType:'application/json',body:JSON.stringify({error:{code:'TEST',message:'保存失败，输入保留'}})})
  const input=req.postDataJSON();state.confirmations.push(input.confirm_replace)
  if(saved&&saved.facts_id!==input.facts_id)state.archived.set(saved.facts_id,copy(saved))
  const row={id:saved?.id??state.saved.size+1,month,...input,version:(saved?.version??0)+1,created_at:saved?.created_at??time,updated_at:time} as MonthlyReview
  state.saved.set(month,row)
  const ids=state.history.get(month)??[];if(!ids.includes(row.facts_id))ids.unshift(row.facts_id);state.history.set(month,ids)
  return reply(document(row.facts_id))
 });return state
}
async function month(page:Page,value:string){const field=page.getByRole('combobox',{name:'回顾月份'});await field.fill(value.slice(0,4)+' 年 '+value.slice(5)+' 月');await field.press('Enter');await field.press('Tab')}
test('按月阅读真实记录，保存失败保留心得，刷新后展示冻结事实和零/未知范围',async({page})=>{
 const state=await mock(page);await page.goto('/life/reviews?month=2026-09')
 await expect(page.getByRole('heading',{name:'我的 2026 年 9 月'})).toBeVisible()
 await expect(page.getByText('本月暂无该账本、币种的账目，不能据此确认收支为零。')).toBeVisible()
 await expect(page.getByText('不完整范围',{exact:true})).toBeVisible();await expect(page.getByText('已填范围差额',{exact:false})).toBeVisible()
 await page.getByLabel('这个月的感受或思考').fill('保留我的感受')
 state.failSave=true;await page.getByRole('button',{name:'保存本月回顾'}).click();await expect(page.getByText('保存失败，输入保留')).toBeVisible();await expect(page.getByLabel('这个月的感受或思考')).toHaveValue('保留我的感受')
 state.failSave=false;await page.getByRole('button',{name:'保存本月回顾'}).click();await expect(page.getByText('已保存的月度档案',{exact:true})).toBeVisible()
 state.title='后来改动的行动';await page.reload();await expect(page.getByRole('heading',{name:'完成行动',exact:true})).toBeVisible();await expect(page.getByLabel('这个月的感受或思考')).toHaveValue('保留我的感受')
 expect(state.saved.size).toBe(1)
})
test('显式重新汇总先预览和确认，保留心得与旧事实版本，来源删除安全提示',async({page})=>{
 const state=await mock(page);await page.goto('/life/reviews?month=2026-09');await page.getByLabel('这个月的感受或思考').fill('原心得');await page.getByRole('button',{name:'保存本月回顾'}).click();await expect(page.getByRole('button',{name:'保存个人回顾'})).toBeVisible()
 state.title='新的事实';await page.getByRole('button',{name:'重新汇总',exact:true}).click();await expect(page.getByRole('heading',{name:'新的事实',exact:true})).toBeVisible();await expect(page.getByLabel('这个月的感受或思考')).toHaveValue('原心得');expect(state.history.get('2026-09')).toHaveLength(1)
 await page.getByRole('button',{name:'确认更新事实并保存回顾'}).click();await page.getByRole('button',{name:'继续预览',exact:true}).click();expect(state.history.get('2026-09')).toHaveLength(1)
 await page.getByRole('button',{name:'确认更新事实并保存回顾'}).click();await page.getByRole('button',{name:'确认更新',exact:true}).click();await expect(page.getByRole('button',{name:'保存个人回顾'})).toBeVisible();expect(state.history.get('2026-09')).toHaveLength(2);expect(state.confirmations).toEqual([false,true])
 await page.locator('.review-history summary').click();await page.locator('.review-history button').last().click();await expect(page.getByRole('heading',{name:'完成行动',exact:true})).toBeVisible();await expect(page.getByText('感受与思考：原心得')).toBeVisible();await page.getByRole('button',{name:'返回当前回顾'}).click()
 state.removed=true;await page.reload();await expect(page.getByText('原始来源已不可用，保留当时摘要。')).toBeVisible();await expect(page.locator('.review-sections section').first().getByRole('link')).toHaveCount(0)
})
test('月份切换跨年、未保存保护、无资产及进行中说明，来源链接正确',async({page})=>{
 await mock(page);await page.goto('/life/reviews?month=2025-12&life_view=goals');await page.getByLabel('这个月最值得记住的事情').fill('未保存')
 await month(page,'2026-01');await page.getByRole('button',{name:'继续填写',exact:true}).click();await expect(page.getByRole('dialog',{name:'未保存的回顾'})).toBeHidden();await expect(page.getByRole('heading',{name:'我的 2025 年 12 月'})).toBeVisible()
 await month(page,'2026-01');await page.getByRole('button',{name:'放弃修改',exact:true}).click();await expect(page.getByRole('heading',{name:'我的 2026 年 1 月'})).toBeVisible();await expect(page.getByLabel('这个月最值得记住的事情')).toHaveValue('');await expect(page.getByText('没有记录资产快照，不推算月末资产。')).toBeVisible()
 await expect(page.locator('.review-sections section').first().getByRole('link',{name:'查看原始记录'})).toHaveAttribute('href','/projects?task=1')
 await expect(page.getByRole('link',{name:'返回生活'})).toHaveAttribute('href','/life?view=goals')
 await month(page,'2026-10');await expect(page.getByText('进行中的月份，记录可能继续增加')).toBeVisible()
})
for(const width of [1440,390])test(`月度回顾 ${width}px 阅读、长金额与心得表单可用`,async({page})=>{
 const state=await mock(page);state.title='一条值得长期保存的行动'.repeat(12);await page.setViewportSize({width,height:1000});await page.goto('/life/reviews?month=2026-09');await expect(page.getByLabel('这个月的感受或思考')).toBeVisible();expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await page.getByLabel('Obsidian 或外部月度笔记（可选）').fill('javascript:alert(1)');await page.getByRole('button',{name:'保存本月回顾'}).click();await expect(page.getByText('链接仅支持 http、https 和有效的 Obsidian open URI。')).toBeVisible();if(process.env.CAPTURE_VISUALS)await page.screenshot({path:`test-results/reviews-${width}.png`,fullPage:true})
})
