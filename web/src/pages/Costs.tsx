import { useState, useEffect } from 'react';
import { fetchJSON, type CostEntry } from '../lib/api';

export function CostsPage() {
  const [costs, setCosts] = useState<{ total: number; breakdown: CostEntry[] }>({ total: 0, breakdown: [] });
  const [metrics, setMetrics] = useState<any>(null);
  const [traces, setTraces] = useState<any[]>([]);
  const [feedbackStats, setFeedbackStats] = useState<any>(null);

  useEffect(() => {
    load();
    const interval = setInterval(load, 5000);
    return () => clearInterval(interval);
  }, []);

  async function load() {
    const [c, m, t, f] = await Promise.all([
      fetchJSON<any>('/api/costs'),
      fetchJSON<any>('/api/metrics?window=60').catch(() => null),
      fetchJSON<any[]>('/api/traces?limit=20').catch(() => []),
      fetchJSON<any>('/api/feedback').catch(() => null),
    ]);
    setCosts(c);
    setMetrics(m);
    setTraces(Array.isArray(t) ? t : []);
    setFeedbackStats(f);
  }

  const maxCost = Math.max(...costs.breakdown.map((b) => b.cost), 0.001);

  return (
    <div className="p-6 w-full overflow-y-auto h-full">
      <h1 className="text-xl font-semibold mb-6">Observability</h1>

      {/* Metrics Cards */}
      <div className="grid grid-cols-5 gap-3 mb-6">
        <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-4">
          <div className="text-[10px] text-[var(--color-text2)] uppercase mb-1">Total Cost</div>
          <div className="text-xl font-bold text-[var(--color-green)]">${costs.total.toFixed(4)}</div>
        </div>
        <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-4">
          <div className="text-[10px] text-[var(--color-text2)] uppercase mb-1">Requests (1h)</div>
          <div className="text-xl font-bold text-[var(--color-accent)]">{metrics?.totalRequests || 0}</div>
        </div>
        <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-4">
          <div className="text-[10px] text-[var(--color-text2)] uppercase mb-1">Avg Latency</div>
          <div className="text-xl font-bold">{metrics?.avgLatencyMs || 0}<span className="text-xs font-normal">ms</span></div>
        </div>
        <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-4">
          <div className="text-[10px] text-[var(--color-text2)] uppercase mb-1">P95 Latency</div>
          <div className="text-xl font-bold">{metrics?.p95LatencyMs || 0}<span className="text-xs font-normal">ms</span></div>
        </div>
        <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-4">
          <div className="text-[10px] text-[var(--color-text2)] uppercase mb-1">Error Rate</div>
          <div className={`text-xl font-bold ${(metrics?.errorRate || 0) > 0.1 ? 'text-[var(--color-red)]' : 'text-[var(--color-green)]'}`}>
            {((metrics?.errorRate || 0) * 100).toFixed(1)}%
          </div>
        </div>
      </div>

      {/* Provider Breakdown + Feedback */}
      <div className="grid grid-cols-2 gap-4 mb-6">
        {/* Provider Metrics */}
        <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-4">
          <div className="text-xs text-[var(--color-text2)] uppercase mb-3">Provider Breakdown (1h)</div>
          {metrics?.byProvider && Object.keys(metrics.byProvider).length > 0 ? (
            <div className="space-y-2">
              {Object.entries(metrics.byProvider).map(([name, pm]: [string, any]) => (
                <div key={name} className="flex items-center justify-between text-sm">
                  <span className="font-medium">{name}</span>
                  <div className="flex gap-4 text-xs text-[var(--color-text2)]">
                    <span>{pm.requests} reqs</span>
                    <span>{pm.avgLatencyMs}ms</span>
                    <span>${pm.cost.toFixed(4)}</span>
                    {pm.errors > 0 && <span className="text-[var(--color-red)]">{pm.errors} err</span>}
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-xs text-[var(--color-text2)]">No data yet</div>
          )}
        </div>

        {/* Feedback / Evals */}
        <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-4">
          <div className="text-xs text-[var(--color-text2)] uppercase mb-3">Response Quality (Evals)</div>
          {feedbackStats && feedbackStats.total > 0 ? (
            <div>
              <div className="flex items-center gap-4 mb-3">
                <div className="text-3xl font-bold">{(feedbackStats.score * 100).toFixed(0)}%</div>
                <div className="text-xs text-[var(--color-text2)]">
                  <div>{feedbackStats.thumbsUp} positive / {feedbackStats.thumbsDown} negative</div>
                  <div>{feedbackStats.total} total ratings</div>
                </div>
              </div>
              {Object.entries(feedbackStats.byModel || {}).map(([model, ms]: [string, any]) => (
                <div key={model} className="flex items-center justify-between text-xs py-1">
                  <span className="font-mono">{model}</span>
                  <div className="flex items-center gap-2">
                    <div className="w-24 h-2 bg-[var(--color-bg)] rounded overflow-hidden">
                      <div className="h-full bg-[var(--color-green)] rounded" style={{ width: `${ms.score * 100}%` }} />
                    </div>
                    <span>{(ms.score * 100).toFixed(0)}%</span>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-xs text-[var(--color-text2)]">
              No feedback yet. Rate responses in chat to start tracking quality.
            </div>
          )}
        </div>
      </div>

      {/* Cost Table */}
      <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl overflow-hidden mb-6">
        <table className="w-full">
          <thead>
            <tr className="bg-[var(--color-surface2)] text-[var(--color-text2)] text-xs uppercase">
              <th className="text-left px-4 py-3">Provider / Model</th>
              <th className="text-left px-4 py-3">Requests</th>
              <th className="text-left px-4 py-3">Tokens</th>
              <th className="text-left px-4 py-3">Cost</th>
              <th className="text-left px-4 py-3">Share</th>
            </tr>
          </thead>
          <tbody>
            {costs.breakdown.length === 0 ? (
              <tr><td colSpan={5} className="text-center py-6 text-[var(--color-text2)] text-sm">No requests yet</td></tr>
            ) : (
              costs.breakdown.map((b) => (
                <tr key={`${b.provider}/${b.model}`} className="border-t border-[var(--color-border)]">
                  <td className="px-4 py-2.5 text-sm">{b.provider}/{b.model}</td>
                  <td className="px-4 py-2.5 text-sm">{b.requests}</td>
                  <td className="px-4 py-2.5 text-sm">{b.tokens.toLocaleString()}</td>
                  <td className="px-4 py-2.5 text-sm">${b.cost.toFixed(4)}</td>
                  <td className="px-4 py-2.5 w-32">
                    <div className="h-4 bg-[var(--color-bg)] rounded overflow-hidden">
                      <div className="h-full bg-[var(--color-accent)] rounded" style={{ width: `${(b.cost / maxCost) * 100}%` }} />
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Request Trace Log */}
      <div className="bg-[var(--color-bg)] border border-[var(--color-border)] rounded-xl p-4 font-mono text-xs max-h-60 overflow-y-auto">
        <div className="text-[var(--color-text2)] uppercase mb-2 font-sans text-xs">Recent Requests</div>
        {traces.length === 0 ? (
          <div className="text-[var(--color-text2)]">No traces yet</div>
        ) : (
          traces.slice().reverse().map((t: any, i: number) => (
            <div key={i} className="flex gap-3 py-0.5 items-center">
              <span className="text-[var(--color-text2)] w-16 shrink-0">
                {new Date(t.timestamp).toLocaleTimeString()}
              </span>
              <span className={`w-12 shrink-0 ${t.status === 'ok' ? 'text-[var(--color-green)]' : 'text-[var(--color-red)]'}`}>
                {t.status}
              </span>
              <span className="text-[var(--color-accent)] w-20 shrink-0 truncate">{t.provider}</span>
              <span className="w-28 shrink-0 truncate">{t.model}</span>
              <span className="text-[var(--color-text2)] w-16 shrink-0">{t.latencyMs}ms</span>
              <span className="text-[var(--color-text2)] w-16 shrink-0">{t.outputTokens}tok</span>
              <span className="text-[var(--color-text2)] w-10 shrink-0">{t.source}</span>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
