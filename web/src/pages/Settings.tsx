import { useState, useEffect } from 'react';
import { fetchJSON, putJSON, type ProviderInfo } from '../lib/api';
import { t } from '../lib/i18n';

export function SettingsPage() {
  const [providers, setProviders] = useState<ProviderInfo[]>([]);
  const [config, setConfig] = useState<{ provider: string; model: string; routerEnabled: boolean } | null>(null);
  const [selProvider, setSelProvider] = useState('');
  const [selModel, setSelModel] = useState('');
  const [responseLang, setResponseLang] = useState('auto');
  const [skillSource, setSkillSource] = useState('all');
  const [saved, setSaved] = useState(false);
  const [mcpServers, setMcpServers] = useState<any[]>([]);

  useEffect(() => {
    fetchJSON<ProviderInfo[]>('/api/providers').then(setProviders);
    fetchJSON<any>('/api/config').then((c) => {
      setConfig(c);
      setSelProvider(c.provider);
      setSelModel(c.model);
      setResponseLang(c.responseLang || 'auto');
    });
    fetchJSON<any[]>('/api/mcp').then(setMcpServers).catch(() => setMcpServers([]));
  }, []);

  const models = providers.find((p) => p.name === selProvider)?.models || [];

  async function apply() {
    await putJSON('/api/config', { provider: selProvider, model: selModel });
    setConfig((c) => c ? { ...c, provider: selProvider, model: selModel } : c);
    setSaved(true);
    setTimeout(() => setSaved(false), 2000);
  }

  async function toggleRouter() {
    const next = !config?.routerEnabled;
    await putJSON('/api/config', { routerEnabled: next });
    setConfig((c) => c ? { ...c, routerEnabled: next } : c);
  }

  return (
    <div className="p-6 w-full max-w-3xl mx-auto">
      <h1 className="text-xl font-semibold mb-6">{t('settings.title')}</h1>

      {/* Provider & Model */}
      <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-6 mb-4">
        <h2 className="text-sm font-semibold mb-4 text-[var(--color-text2)] uppercase tracking-wide">{t('settings.defaultProvider')}</h2>

        {config && (
          <div className="text-xs text-[var(--color-text2)] mb-4 px-3 py-2 bg-[var(--color-bg)] rounded-lg">
            Current: <span className="text-[var(--color-green)] font-medium">{config.provider} / {config.model}</span>
          </div>
        )}

        <div className="grid grid-cols-2 gap-4 mb-4">
          <div>
            <label className="block text-xs text-[var(--color-text2)] mb-1.5 uppercase">Provider</label>
            <select
              value={selProvider}
              onChange={(e) => { setSelProvider(e.target.value); setSelModel(''); }}
              className="w-full bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg px-3 py-2.5 text-sm text-[var(--color-text)]"
            >
              {providers.map((p) => (
                <option key={p.name} value={p.name}>{p.displayName}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-xs text-[var(--color-text2)] mb-1.5 uppercase">Model</label>
            <select
              value={selModel}
              onChange={(e) => setSelModel(e.target.value)}
              className="w-full bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg px-3 py-2.5 text-sm text-[var(--color-text)]"
            >
              <option value="">Select...</option>
              {models.map((m) => (
                <option key={m.id} value={m.id}>{m.displayName} ({m.id})</option>
              ))}
            </select>
          </div>
        </div>

        <button
          onClick={apply}
          className={`w-full py-2.5 rounded-lg text-sm font-medium transition-colors ${
            saved ? 'bg-[var(--color-green)] text-white' : 'bg-[var(--color-accent)] text-white hover:bg-[var(--color-accent2)]'
          }`}
        >
          {saved ? t('settings.applied') : t('settings.apply')}
        </button>
      </div>

      {/* Toggles */}
      <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-6">
        <h2 className="text-sm font-semibold mb-4 text-[var(--color-text2)] uppercase tracking-wide">{t('settings.options')}</h2>

        <div className="flex items-center justify-between py-3 border-b border-[var(--color-border)]">
          <div>
            <div className="text-sm font-medium">{t('settings.smartRouter')}</div>
            <div className="text-xs text-[var(--color-text2)]">{t('settings.smartRouterDesc')}</div>
          </div>
          <button
            onClick={toggleRouter}
            className={`w-11 h-6 rounded-full transition-colors relative ${
              config?.routerEnabled ? 'bg-[var(--color-accent)]' : 'bg-[var(--color-surface2)]'
            }`}
          >
            <div className={`w-4 h-4 bg-white rounded-full absolute top-1 transition-transform ${
              config?.routerEnabled ? 'translate-x-6' : 'translate-x-1'
            }`} />
          </button>
        </div>

        <div className="flex items-center justify-between py-3 border-b border-[var(--color-border)]">
          <div>
            <div className="text-sm font-medium">{t('settings.authPass')}</div>
            <div className="text-xs text-[var(--color-text2)]">{t('settings.authPassDesc')}</div>
          </div>
          <div className="w-11 h-6 rounded-full bg-[var(--color-accent)] relative">
            <div className="w-4 h-4 bg-white rounded-full absolute top-1 translate-x-6" />
          </div>
        </div>

        {/* Response Language */}
        <div className="py-3">
          <div className="text-sm font-medium mb-2">{t('settings.language')}</div>
          <div className="text-xs text-[var(--color-text2)] mb-3">AI 응답 언어 / AI response language</div>
          <div className="flex gap-2 flex-wrap">
            {[
              { id: 'auto', label: '🌐 Auto (자동감지)', desc: 'Follow user language' },
              { id: 'ko', label: '🇰🇷 한국어', desc: 'Korean' },
              { id: 'en', label: '🇺🇸 English', desc: 'English' },
              { id: 'ja', label: '🇯🇵 日本語', desc: 'Japanese' },
              { id: 'zh', label: '🇨🇳 中文', desc: 'Chinese' },
            ].map((lang) => (
              <button
                key={lang.id}
                onClick={async () => {
                  setResponseLang(lang.id);
                  await putJSON('/api/config', { responseLang: lang.id });
                }}
                className={`px-4 py-2 rounded-lg text-sm transition-colors ${
                  responseLang === lang.id
                    ? 'bg-[var(--color-accent)] text-white'
                    : 'bg-[var(--color-surface2)] text-[var(--color-text2)] hover:text-[var(--color-text)]'
                }`}
              >
                {lang.label}
              </button>
            ))}
          </div>
        </div>

        {/* Skill Source */}
        <div className="py-3 border-t border-[var(--color-border)]">
          <div className="text-sm font-medium mb-2">Skill Source</div>
          <div className="text-xs text-[var(--color-text2)] mb-3">스킬을 어디서 가져올지 / Where to load skills from</div>
          <div className="flex gap-2 flex-wrap">
            {[
              { id: 'all', label: 'All (전체)' },
              { id: 'claude', label: 'Claude Code' },
              { id: 'codex', label: 'Codex CLI' },
              { id: 'gemini', label: 'Gemini CLI' },
              { id: 'none', label: 'None (없음)' },
            ].map((src) => (
              <button
                key={src.id}
                onClick={async () => {
                  setSkillSource(src.id);
                  await putJSON('/api/skill-source', { source: src.id });
                }}
                className={`px-4 py-2 rounded-lg text-sm transition-colors ${
                  skillSource === src.id
                    ? 'bg-[var(--color-accent)] text-white'
                    : 'bg-[var(--color-surface2)] text-[var(--color-text2)] hover:text-[var(--color-text)]'
                }`}
              >
                {src.label}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* MCP Servers */}
      <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-6 mt-4">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-sm font-semibold text-[var(--color-text2)] uppercase tracking-wide">MCP Servers</h2>
          <button
            onClick={async () => {
              await fetchJSON('/api/mcp/connect', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({}) });
              const data = await fetchJSON<any[]>('/api/mcp');
              setMcpServers(data);
            }}
            className="text-xs px-3 py-1.5 bg-[var(--color-accent)] text-white rounded-lg hover:opacity-80"
          >
            Reconnect
          </button>
        </div>
        {mcpServers.length === 0 ? (
          <div className="text-xs text-[var(--color-text2)]">
            No MCP servers configured. Add servers in <code className="bg-[var(--color-bg)] px-1 rounded">.mcp.json</code> or <code className="bg-[var(--color-bg)] px-1 rounded">.claude/settings.json</code>
          </div>
        ) : (
          <div className="space-y-2">
            {mcpServers.map((srv: any, i: number) => (
              <div key={i} className="flex items-center justify-between bg-[var(--color-bg)] rounded-lg px-3 py-2">
                <div>
                  <div className="text-sm font-medium">{srv.name || `Server ${i + 1}`}</div>
                  <div className="text-[10px] text-[var(--color-text2)]">{srv.command || srv.url || '—'}</div>
                </div>
                <div className="flex items-center gap-2">
                  <span className={`text-[10px] px-1.5 py-0.5 rounded ${srv.connected ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'}`}>
                    {srv.connected ? 'Connected' : 'Disconnected'}
                  </span>
                  <span className="text-[10px] text-[var(--color-text2)]">{srv.tools || 0} tools</span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
