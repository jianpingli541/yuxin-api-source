package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// StatusPageHTML 公开状态页（HTML）
func StatusPageHTML(c *gin.Context) {
	html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>豫鑫 API · 服务状态</title>
    <link rel="icon" href="/logo.svg" type="image/svg+xml">
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0f172a; color: #e2e8f0; min-height: 100vh; }
        .container { max-width: 900px; margin: 0 auto; padding: 2rem 1rem; }
        .header { text-align: center; margin-bottom: 3rem; }
        .header h1 { font-size: 2rem; color: #f8fafc; margin-bottom: 0.5rem; }
        .header p { color: #94a3b8; font-size: 0.95rem; }
        .status-banner { background: #1e293b; border-radius: 12px; padding: 1.5rem; margin-bottom: 2rem; border: 1px solid #334155; display: flex; align-items: center; gap: 1rem; }
        .status-dot { width: 12px; height: 12px; border-radius: 50%; animation: pulse 2s infinite; }
        .status-dot.operational { background: #22c55e; box-shadow: 0 0 12px #22c55e; }
        .status-dot.partial { background: #eab308; box-shadow: 0 0 12px #eab308; }
        .status-dot.major { background: #ef4444; box-shadow: 0 0 12px #ef4444; }
        @keyframes pulse { 0%,100% { opacity: 1; } 50% { opacity: 0.5; } }
        .status-text { font-size: 1.1rem; font-weight: 600; }
        .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 1rem; margin-bottom: 2rem; }
        .stat-card { background: #1e293b; border-radius: 8px; padding: 1.2rem; border: 1px solid #334155; text-align: center; }
        .stat-value { font-size: 1.8rem; font-weight: 700; color: #f8fafc; }
        .stat-label { font-size: 0.8rem; color: #64748b; margin-top: 0.3rem; }
        .channel-item { background: #1e293b; border-radius: 8px; padding: 1rem; border: 1px solid #334155; margin-bottom: 0.5rem; display: flex; justify-content: space-between; }
        .ch-dot { width: 8px; height: 8px; border-radius: 50%; display: inline-block; margin-right: 8px; }
        .footer { text-align: center; padding: 2rem 0; color: #475569; font-size: 0.85rem; }
        .footer a { color: #3b82f6; text-decoration: none; }
    </style>
</head>
<body><div class="container"><div class="header"><h1>豫鑫 API 服务状态</h1><p>实时监控所有渠道运行状态</p></div>
<div class="status-banner"><div class="status-dot operational" id="dot"></div><div><div class="status-text" id="status">加载中...</div></div></div>
<div class="stats-grid"><div class="stat-card"><div class="stat-value" id="total">-</div><div class="stat-label">总渠道</div></div>
<div class="stat-card"><div class="stat-value" id="ok">-</div><div class="stat-label">正常</div></div>
<div class="stat-card"><div class="stat-value" id="bad">-</div><div class="stat-label">异常</div></div></div>
<div id="channels"></div>
<div class="footer"><p><a href="/">返回首页</a> · <a href="/pricing-page">查看定价</a></p></div></div>
<script>
fetch('/api/status_page').then(r=>r.json()).then(d=>{
if(!d.success)return;
const data=d.data;
const map={operational:'所有系统正常运行',partial_outage:'部分服务降级',major_outage:'服务中断'};
document.getElementById('status').textContent=map[data.overall_status]||data.overall_status;
document.getElementById('dot').className='status-dot '+(data.overall_status==='operational'?'operational':data.overall_status==='partial_outage'?'partial':'major');
document.getElementById('total').textContent=data.total_channels;
document.getElementById('ok').textContent=data.healthy;
document.getElementById('bad').textContent=data.down;
const sm={operational:'#22c55e',degraded:'#eab308',down:'#ef4444',unknown:'#64748b'};
const st={operational:'正常',degraded:'降级',down:'故障',unknown:'未知'};
document.getElementById('channels').innerHTML=(data.channels||[]).map(c=>
'<div class="channel-item"><div><span class="ch-dot" style="background:'+sm[c.status]+'"></span>'+c.name+'<br><small style="color:#64748b">'+(c.models||[]).slice(0,3).join(', ')+'</small></div><span style="color:#94a3b8">'+(st[c.status]||c.status)+(c.response_time_ms>0?' '+c.response_time_ms+'ms':'')+'</span></div>'
).join('');
});
</script></body></html>`
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// PricingPageHTML 公开定价页（HTML）
func PricingPageHTML(c *gin.Context) {
	html := `<!DOCTYPE html>
<html lang="zh-CN"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>豫鑫 API · 模型定价</title><link rel="icon" href="/logo.svg" type="image/svg+xml">
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;background:#0f172a;color:#e2e8f0}
.container{max-width:1000px;margin:0 auto;padding:2rem 1rem}
.header{text-align:center;margin-bottom:2rem}
.header h1{font-size:2rem;color:#f8fafc}
.header p{color:#94a3b8}
.search{width:100%;padding:0.75rem 1rem;border-radius:8px;border:1px solid #334155;background:#1e293b;color:#e2e8f0;font-size:0.95rem;outline:none;margin-bottom:1rem}
.grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(280px,1fr));gap:1rem}
.card{background:#1e293b;border-radius:12px;padding:1.2rem;border:1px solid #334155}
.card:hover{border-color:#3b82f6}
.name{font-size:1.1rem;font-weight:600;color:#f8fafc;margin-bottom:0.5rem}
.price{display:flex;justify-content:space-between;border-top:1px solid #334155;padding-top:0.5rem;margin-top:0.5rem}
.price-val{color:#4ade80;font-weight:600;font-family:monospace}
.cap{font-size:0.7rem;padding:2px 6px;border-radius:3px;background:#1e3a5f;color:#93c5fd;display:inline-block;margin:0 2px 2px 0}
.footer{text-align:center;padding:2rem 0;color:#475569;font-size:0.85rem}
.footer a{color:#3b82f6;text-decoration:none}
</style></head><body><div class="container">
<div class="header"><h1>模型定价</h1><p>透明定价 · 按量计费</p></div>
<input class="search" placeholder="搜索模型..." oninput="filter(this.value)">
<div class="grid" id="grid">加载中...</div>
<div class="footer"><p><a href="/">首页</a> · <a href="/status-page">服务状态</a></p></div>
</div>
<script>
let models=[];
fetch('/api/public/pricing').then(r=>r.json()).then(d=>{
if(!d.success)return;
models=d.data.models;
render(models);
});
function filter(q){
q=q.toLowerCase();
render(models.filter(m=>m.id.toLowerCase().includes(q)));
}
function render(list){
if(!list.length){document.getElementById('grid').innerHTML='<p style="color:#64748b;text-align:center">未找到模型</p>';return;}
document.getElementById('grid').innerHTML=list.map(m=>{
const caps=(m.capabilities||[]).map(c=>'<span class="cap">'+c+'</span>').join('');
return '<div class="card"><div class="name">'+m.id+'</div>'+caps+
'<div class="price"><span>输入</span><span class="price-val">$'+fmt(m.pricing.prompt)+'/1K</span></div>'+
'<div class="price"><span>输出</span><span class="price-val">$'+fmt(m.pricing.completion)+'/1K</span></div>'+
(m.pricing.cache_read?'<div class="price"><span>缓存</span><span class="price-val">$'+fmt(m.pricing.cache_read)+'/1K</span></div>':'')+
'</div>';
}).join('');
}
function fmt(p){if(!p||p==='0')return '0';const n=parseFloat(p);return n<0.001?n.toFixed(6):n<1?n.toFixed(4):n.toFixed(2);}
</script></body></html>`
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
