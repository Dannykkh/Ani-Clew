import { useState, useEffect } from 'react';
import { fetchJSON, type CostEntry } from '../lib/api';

export function CostsPage() {
  const [costs, setCosts] = useState<{ total: number; breakdown: CostEntry[] }>({ total: 0, breakdown: [] });

  useEffect(() => {
    const load = () => fetchJSON<any>('/api/costs').then(setCosts);
    load();
    const interval = setInterval(load, 5000);
    return () => clearInterval(interval);
  }, []);

  const maxCost = Math.max(...costs.breakdown.map((b) => b.cost), 0.001);

  return (
    <div className="p-6">
      <h1 className="text-xl font-semibold mb-6">Cost Breakdown</h1>

      {/* Summary Cards */}
      <div className="grid grid-cols-3 gap-4 mb-6">
        <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-5">
          <div className="text-xs text-[var(--color-text2)] uppercase mb-1">Total Cost</div>
          <div className="text-2xl font-bold text-[var(--color-green)]">${costs.total.toFixed(4)}</div>
        </div>
        <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-5">
          <div className="text-xs text-[var(--color-text2)] uppercase mb-1">Requests</div>
          <div className="text-2xl font-bold text-[var(--color-accent)]">
            {costs.breakdown.reduce((s, b) => s + b.requests, 0)}
          </div>
        </div>
        <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-5">
          <div className="text-xs text-[var(--color-text2)] uppercase mb-1">Models Used</div>
          <div className="text-2xl font-bold">{costs.breakdown.length}</div>
        </div>
      </div>

      {/* Table */}
      <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl overflow-hidden">
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
              <tr><td colSpan={5} className="text-center py-8 text-[var(--color-text2)] text-sm">No requests yet</td></tr>
            ) : (
              costs.breakdown.map((b) => (
                <tr key={`${b.provider}/${b.model}`} className="border-t border-[var(--color-border)]">
                  <td className="px-4 py-3 text-sm">{b.provider}/{b.model}</td>
                  <td className="px-4 py-3 text-sm">{b.requests}</td>
                  <td className="px-4 py-3 text-sm">{b.tokens.toLocaleString()}</td>
                  <td className="px-4 py-3 text-sm">${b.cost.toFixed(4)}</td>
                  <td className="px-4 py-3 w-40">
                    <div className="h-5 bg-[var(--color-bg)] rounded overflow-hidden">
                      <div
                        className="h-full bg-[var(--color-accent)] rounded transition-all"
                        style={{ width: `${(b.cost / maxCost) * 100}%` }}
                      />
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
