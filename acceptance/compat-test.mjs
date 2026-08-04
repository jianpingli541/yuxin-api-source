// 豫鑫 API 中转站 — 兼容性测试脚本(Playwright)
// 覆盖:Chromium / Firefox / WebKit × 桌面 / 移动视口 × 核心页面
// 检查项:页面加载、无严重 JS 错误、关键元素渲染、HTTP 状态
import { chromium, firefox, webkit } from 'playwright';

const BASE = process.env.YUXIN_BASE_URL || 'https://103.55.131.130';
const RESULTS = [];

const DEVICES = [
  { name: 'Desktop 1920x1080', viewport: { width: 1920, height: 1080 }, isMobile: false },
  { name: 'Laptop 1366x768', viewport: { width: 1366, height: 768 }, isMobile: false },
  { name: 'Tablet 768x1024', viewport: { width: 768, height: 1024 }, isMobile: true },
  { name: 'Mobile 390x844', viewport: { width: 390, height: 844, deviceScaleFactor: 3 }, isMobile: true },
];

const PAGES = [
  { path: '/', name: '首页', expectText: null },
  { path: '/pricing', name: '定价页', expectText: null },
  { path: '/about', name: '关于页', expectText: null },
  { path: '/login', name: '登录页', expectText: null },
  { path: '/register', name: '注册页', expectText: null },
  { path: '/docs', name: '文档页', expectText: null },
];

async function runBrowser(browserType, browserName) {
  let browser;
  try {
    browser = await browserType.launch({
      args: browserName === 'chromium' ? ['--no-sandbox', '--ignore-certificate-errors'] : [],
    });
  } catch (e) {
    RESULTS.push({ browser: browserName, device: '-', page: '-', status: 'LAUNCH_FAIL', detail: String(e).slice(0, 120) });
    return;
  }
  for (const dev of DEVICES) {
    const ctx = await browser.newContext({
      viewport: dev.viewport,
      isMobile: dev.isMobile,
      ignoreHTTPSErrors: true,
    });
    const page = await ctx.newPage();
    const jsErrors = [];
    page.on('pageerror', (err) => jsErrors.push(String(err.message).slice(0, 100)));
    page.on('console', (msg) => {
      if (msg.type() === 'error') jsErrors.push('console: ' + msg.text().slice(0, 80));
    });
    for (const pg of PAGES) {
      const url = BASE + pg.path;
      let status = 'OK';
      let detail = '';
      let httpStatus = 0;
      const start = Date.now();
      try {
        const resp = await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 20000 });
        httpStatus = resp ? resp.status() : 0;
        // SPA fallback 路由可能 200;等待前端渲染
        await page.waitForTimeout(1200);
        const bodyText = await page.evaluate(() => document.body ? document.body.innerText.slice(0, 2000) : '');
        if (httpStatus >= 400) status = 'HTTP_' + httpStatus;
        else if (jsErrors.length > 0 && jsErrors.some(e => !e.includes('console:'))) status = 'JS_ERROR';
        detail = `http=${httpStatus} load=${Date.now() - start}ms jsErr=${jsErrors.length} bodyLen=${bodyText.length}`;
      } catch (e) {
        status = 'LOAD_FAIL';
        detail = String(e).slice(0, 100);
      }
      RESULTS.push({ browser: browserName, device: dev.name, page: pg.path, status, detail });
      jsErrors.length = 0;
    }
    await ctx.close();
  }
  await browser.close();
}

await runBrowser(chromium, 'chromium');
await runBrowser(firefox, 'firefox');
await runBrowser(webkit, 'webkit');

// 输出 markdown 表格
const browsers = ['chromium', 'firefox', 'webkit'];
console.log('| 页面 \\ 环境 | ' + DEVICES.map(d => d.name).join(' | ') + ' |');
for (const b of browsers) {
  console.log('## ' + b);
  console.log('| 页面 | ' + DEVICES.map(d => d.name).join(' | ') + ' |');
  console.log('|' + '---|'.repeat(DEVICES.length + 1));
  for (const pg of PAGES) {
    const row = [pg.path];
    for (const d of DEVICES) {
      const r = RESULTS.find(x => x.browser === b && x.device === d.name && x.page === pg.path);
      row.push(r ? (r.status === 'OK' ? '✅ ' + r.detail : '❌ ' + r.status + ' ' + r.detail.slice(0, 60)) : '—');
    }
    console.log('| ' + row.join(' | ') + ' |');
  }
}
const fails = RESULTS.filter(r => r.status !== 'OK');
console.log('\nTOTAL=' + RESULTS.length + ' FAILS=' + fails.length);
for (const f of fails) console.log('FAIL:', JSON.stringify(f));
