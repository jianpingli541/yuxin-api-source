import { useState } from 'react'
import { PublicLayout } from '@/components/layout'
import { PageTransition } from '@/components/page-transition'

const API_ENDPOINTS = [
  { method: 'POST', path: '/v1/chat/completions', desc: '对话补全', category: 'OpenAI' },
  { method: 'GET', path: '/v1/models', desc: '模型列表', category: 'OpenAI' },
  { method: 'POST', path: '/v1/images/generations', desc: '图像生成', category: 'OpenAI' },
  { method: 'POST', path: '/v1/audio/speech', desc: '文本转语音', category: 'OpenAI' },
  { method: 'POST', path: '/v1/embeddings', desc: '文本嵌入', category: 'OpenAI' },
  { method: 'POST', path: '/v1/messages', desc: 'Claude 消息 API', category: 'Anthropic' },
  { method: 'POST', path: '/v1beta/models/:generateContent', desc: 'Gemini 生成', category: 'Gemini' },
]

const CODE_EXAMPLES: Record<string, string> = {
  curl: `curl -X POST https://your-domain.com/v1/chat/completions \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -d '{"model":"gpt-4o","messages":[{"role":"user","content":"Hello!"}]}'`,
  python: `from openai import OpenAI

client = OpenAI(api_key="sk-your-key", base_url="https://your-domain.com/v1")
resp = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role":"user","content":"Hello!"}],
)
print(resp.choices[0].message.content)`,
  node: `import OpenAI from "openai";
const client = new OpenAI({ apiKey:"sk-your-key", baseURL:"https://your-domain.com/v1" });
const resp = await client.chat.completions.create({
  model: "gpt-4o",
  messages: [{ role:"user", content:"Hello!" }],
});
console.log(resp.choices[0].message.content);`,
}

const TABS = ['curl','python','node'] as const

export function DocsPage() {
  const [tab, setTab] = useState<(typeof TABS)[number]>('curl')
  return (
    <PublicLayout><PageTransition>
      <div className="mx-auto max-w-5xl px-6 py-12">
        <h1 className="text-3xl font-bold mb-2">API 文档</h1>
        <p className="text-muted-foreground mb-8">兼容 OpenAI / Anthropic / Gemini 格式，可直接使用官方 SDK。</p>
        <section className="mb-12">
          <h2 className="text-xl font-semibold mb-4">快速开始</h2>
          <div className="flex gap-2 mb-4">
            {TABS.map(t => (
              <button key={t} onClick={() => setTab(t)}
                className={`px-4 py-2 rounded-lg text-sm font-medium ${tab===t?'bg-primary text-primary-foreground':'bg-muted text-muted-foreground hover:bg-accent'}`}>
                {t==='node'?'Node.js':t[0].toUpperCase()+t.slice(1)}
              </button>
            ))}
          </div>
          <pre className="bg-zinc-950 text-zinc-100 rounded-xl p-6 overflow-x-auto text-sm"><code>{CODE_EXAMPLES[tab]}</code></pre>
        </section>
        <section className="mb-12">
          <h2 className="text-xl font-semibold mb-4">API 端点</h2>
          <div className="space-y-2">
            {API_ENDPOINTS.map((ep,i) => (
              <div key={i} className="flex items-center gap-3 rounded-lg border border-border/40 p-4 hover:bg-accent/30">
                <span className={`px-2 py-1 rounded text-xs font-bold w-16 text-center ${ep.method==='GET'?'bg-blue-500/20 text-blue-600':'bg-green-500/20 text-green-600'}`}>{ep.method}</span>
                <code className="text-sm font-mono">{ep.path}</code>
                <span className="text-muted-foreground text-sm flex-1">{ep.desc}</span>
                <span className="px-2 py-0.5 rounded text-xs bg-muted text-muted-foreground">{ep.category}</span>
              </div>
            ))}
          </div>
        </section>
      </div>
    </PageTransition></PublicLayout>
  )
}
export const Component = DocsPage
export const action = async () => {}
// Attach displayName for React DevTools (TanStack Router route component)
;(Component as React.FunctionComponent).displayName = 'DocsPage'
