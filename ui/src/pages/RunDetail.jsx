export default function RunDetail({ run, onClose }) {
  if (!run) return null
  return (
    <div className="card">
      <div className="flex items-center justify-between mb-4">
        <div>
          <p className="text-sm uppercase tracking-[0.2em] text-slate-500">Run detail</p>
          <h2 className="text-2xl font-semibold">{run.target}</h2>
        </div>
        <button className="text-sm text-slate-500 hover:text-slate-800" onClick={onClose}>
          Close
        </button>
      </div>
      <div className="grid gap-3 md:grid-cols-3">
        <div className="p-3 rounded-xl bg-slate-900 text-white">
          <p className="text-xs text-slate-300">Score</p>
          <p className="text-3xl font-semibold">{Math.round(run.score || 0)}</p>
        </div>
        <div className="p-3 rounded-xl bg-white border border-slate-200">
          <p className="text-xs text-slate-500">Summary</p>
          <p className="font-medium text-slate-800">{run.summary}</p>
        </div>
        <div className="p-3 rounded-xl bg-white border border-slate-200">
          <p className="text-xs text-slate-500">Timestamp</p>
          <p className="font-medium text-slate-800">{new Date(run.timestamp).toLocaleString()}</p>
        </div>
      </div>
      <div className="mt-4">
        <p className="text-sm font-semibold text-slate-700">Probes</p>
        <div className="mt-2 grid gap-2 md:grid-cols-2">
          {(run.probes || []).map((p) => (
            <div key={p.name} className="p-3 rounded-lg border border-slate-200 bg-slate-50">
              <div className="flex items-center justify-between">
                <span className="font-medium">{p.name}</span>
                <span
                  className={`text-xs px-2 py-1 rounded-full ${
                    p.status?.toLowerCase() === 'ok'
                      ? 'bg-emerald-100 text-emerald-700'
                      : p.status?.toLowerCase() === 'na'
                      ? 'bg-slate-200 text-slate-700'
                      : 'bg-rose-100 text-rose-700'
                  }`}
                >
                  {p.status}
                </span>
              </div>
              {p.error && <p className="text-xs text-rose-600 mt-1">{p.error}</p>}
              {p.latency_ms && <p className="text-xs text-slate-500">Latency: {p.latency_ms.toFixed(1)} ms</p>}
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
