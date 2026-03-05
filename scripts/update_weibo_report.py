#!/usr/bin/env python3
import json
import os
import re
import sys
import time
from datetime import datetime, timezone
from urllib.parse import quote
from urllib.request import Request, urlopen

TIANAPI_KEY = os.getenv("TIANAPI_KEY", "e44d0cd4f0b667be2d22cb9130214590")
BRAVE_API_KEY = os.getenv("BRAVE_API_KEY")
ROOT = os.path.dirname(os.path.dirname(__file__))
OUT_JSON = os.path.join(ROOT, "weibo-hot-data.json")
OUT_HTML = os.path.join(ROOT, "public", "weibo-hot-analysis-report.html")


def get_json(url, headers=None):
    req = Request(url, headers=headers or {})
    with urlopen(req, timeout=25) as r:
        return json.loads(r.read().decode("utf-8", errors="ignore"))


def clean_hotnum(raw):
    m = re.findall(r"\d+", str(raw or ""))
    return int(m[-1]) if m else 0


def fetch_weibo_top10():
    data = get_json(f"https://apis.tianapi.com/weibohot/index?key={TIANAPI_KEY}")
    if data.get("code") != 200:
        raise RuntimeError(f"tianapi failed: {data}")
    items = data.get("result", {}).get("list", [])[:10]
    return [{
        "rank": i + 1,
        "hotword": it.get("hotword", ""),
        "hotwordnum": clean_hotnum(it.get("hotwordnum", "")),
        "hottag": it.get("hottag", "") or "",
    } for i, it in enumerate(items)]


def brave_search(query):
    if not BRAVE_API_KEY:
        return None
    url = "https://api.search.brave.com/res/v1/web/search?q=" + quote(query)
    headers = {"Accept": "application/json", "X-Subscription-Token": BRAVE_API_KEY}

    # 频率控制：每次请求前等 1 秒，避免超过 1 req/s
    time.sleep(1.0)
    for attempt in range(2):  # 最多重试一次
        try:
            data = get_json(url, headers=headers)
            web = data.get("web", {}).get("results", [])
            snippets = [(r.get("description") or "").strip() for r in web[:3]]
            snippets = [s for s in snippets if s]
            if not snippets:
                return None
            return {"summary": " ".join(snippets)[:360], "source_title": web[0].get("title", "")}
        except Exception:
            if attempt == 0:
                time.sleep(1.0)
                continue
            return None


def score_and_idea(topic, found):
    heat = topic["hotwordnum"]
    fun = max(35, min(80, 45 + heat // 100000))
    useful = max(8, min(20, 12 if any(k in topic["hotword"] for k in ["中国", "政府", "建议", "芯片"]) else 9))
    total = int(fun + useful)
    return {
        "name": f"{topic['hotword'][:8]}·灵感雷达",
        "pitch": "把热搜事件自动拆成可执行产品机会卡，半小时一更。",
        "inspiration": f"灵感来源：检索结果显示“{found['source_title']}”及相关报道持续讨论该话题，热度值约 {heat}，适合做实时机会发现。",
        "features": ["话题脉络 3 句摘要", "一键生成产品机会卡与评分", "按热度与评分排序筛选"],
        "targetUsers": "产品经理、增长运营、内容团队",
        "funScore": int(fun),
        "usefulScore": int(useful),
        "totalScore": total,
    }


def build(topics):
    rows = []
    ok = 0
    for t in topics:
        found = brave_search(t["hotword"])
        if not found:
            rows.append({**t, "summary": "搜索未获取到有效信息", "product": None})
            continue
        ok += 1
        rows.append({**t, "summary": found["summary"], "product": score_and_idea(t, found)})
    return rows, ok, len(topics) - ok


def render_html(data):
    data_js = json.dumps(data, ensure_ascii=False)
    now = datetime.now(timezone.utc).strftime("%Y-%m-%d %H:%M UTC")
    tpl = '''<!DOCTYPE html>
<html lang="zh-CN"><head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>微博热搜产品创意报告</title>
<style>
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"PingFang SC",sans-serif;background:#f8f9fa;color:#2c3e50;margin:0}
.container{max-width:960px;margin:0 auto;padding:20px}.card{background:#fff;border:1px solid #e5e7eb;border-radius:12px;padding:14px;margin:12px 0}
.tier-top{border-color:#e74c3c;box-shadow:0 0 0 2px rgba(231,76,60,.12)}.tier-mid{border-color:#27ae60}.tier-low{border-color:#bdc3c7}
.muted{color:#6b7280}.hot{display:inline-block;background:#fee2e2;color:#b91c1c;padding:2px 8px;border-radius:999px;font-size:12px}
.nav{position:sticky;top:0;background:#fff;padding:8px 0;display:flex;gap:8px;flex-wrap:wrap;border-bottom:1px solid #eee}
button{border:1px solid #ddd;background:#fff;padding:6px 10px;border-radius:8px;cursor:pointer}
.score{height:8px;background:#eee;border-radius:999px;overflow:hidden}.bar{height:100%;width:0;background:#e74c3c;transition:width .8s}
</style></head><body><div class="container">
<h1>微博热搜产品创意报告</h1><p class="muted">更新时间：__NOW__</p><div id="stats" class="muted"></div>
<div class="nav"><button data-sort="rank">按热搜排名</button><button data-sort="score">按总分</button><button data-filter="all">全部</button><button data-filter="top">优秀(≥80)</button><button data-filter="mid">良好(60-79)</button><button data-filter="low">一般(<60)</button></div>
<div id="cards"></div></div>
<script>
const DATA = __DATA__;
let sortBy='rank', filterBy='all'; const cards=document.getElementById('cards');
function tier(s){if(s>=80)return 'top';if(s>=60)return 'mid';return 'low'}
function pass(item){if(filterBy==='all')return true;if(!item.product)return filterBy==='low';return tier(item.product.totalScore)===filterBy}
function render(){
 const list=DATA.filter(pass).sort((a,b)=>sortBy==='score'?((b.product?.totalScore||0)-(a.product?.totalScore||0)):a.rank-b.rank);
 const valid=DATA.filter(x=>x.product); const avg=valid.reduce((s,x)=>s+x.product.totalScore,0)/Math.max(1,valid.length);
 document.getElementById('stats').textContent=`总话题 ${DATA.length} · 有效分析 ${valid.length} · 平均分 ${avg.toFixed(1)}`;
 cards.innerHTML=list.map(x=>{const p=x.product;const score=p?.totalScore||0;const cls=p?(score>=80?'tier-top':(score>=60?'tier-mid':'tier-low')):'tier-low';
  return `<div class="card ${cls}"><div>#${x.rank} ${x.hotword} ${x.hottag?`<span class='hot'>${x.hottag}</span>`:''}</div><div class='muted'>热度：${x.hotwordnum}</div><p>${x.summary}</p>${p?`<p><b>${p.name}</b>：${p.pitch}</p><p class='muted'>${p.inspiration}</p><p>功能：${p.features.join(' / ')}</p><p>目标用户：${p.targetUsers}</p><div class='score'><div class='bar' data-w='${score}'></div></div><div class='muted'>总分 ${score}（趣味 ${p.funScore} + 实用 ${p.usefulScore}）</div>`:'<p class="muted">未生成产品创意</p>'}</div>`
 }).join('');
 const io=new IntersectionObserver(es=>es.forEach(e=>{if(e.isIntersecting)e.target.style.width=e.target.dataset.w+'%'}),{threshold:.3});
 document.querySelectorAll('.bar').forEach(b=>io.observe(b));
}
document.querySelectorAll('[data-sort]').forEach(b=>b.onclick=()=>{sortBy=b.dataset.sort;render()});
document.querySelectorAll('[data-filter]').forEach(b=>b.onclick=()=>{filterBy=b.dataset.filter;render()});render();
</script></body></html>'''
    return tpl.replace("__DATA__", data_js).replace("__NOW__", now)


def main():
    topics = fetch_weibo_top10()
    data, ok, skip = build(topics)
    with open(OUT_JSON, "w", encoding="utf-8") as f:
        json.dump(data, f, ensure_ascii=False, indent=2)
    with open(OUT_HTML, "w", encoding="utf-8") as f:
        f.write(render_html(data))
    print(f"done. success={ok}, skipped={skip}")
    print(OUT_JSON)
    print(OUT_HTML)


if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        print(f"ERROR: {e}", file=sys.stderr)
        sys.exit(1)
