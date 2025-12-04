import { useEffect, useState } from 'react'

const demoRuns = [
  {
    id: 'demo-1',
    target: '8.8.8.8',
    summary: 'Healthy — minor jitter',
    score: 92,
    timestamp: '2024-01-01T00:00:00Z',
  },
  {
    id: 'demo-2',
    target: '1.1.1.1',
    summary: 'Loss on last mile',
    score: 48,
    timestamp: '2024-01-01T02:10:00Z',
  },
]

export default function Home({ onSelectRun }) {
  const [runs, setRuns] = useState(demoRuns)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    const fetchRuns = async () => {
      setLoading(true)
      try {
        const resp = await fetch('http://localhost:8000/runs')
        if (resp.ok) {
          const data = await resp.json()
          if (Array.isArray(data)) {
            setRuns(data)
          }
        }
      } catch (err) {
        console.warn('Backend unavailable, falling back to demo data', err)
      } finally {
        setLoading(false)
      }
    }
    fetchRuns()
  }, [])

  return (
    <section className="space-y-4">
      <div className="card">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-slate-500 text-sm">Recent runs</p>
            <p className="text-lg font-semibold text-slate-900">End-to-end probe results</p>
          </div>
          <button
            className="px-4 py-2 rounded-full bg-black text-white hover:translate-y-[-1px] transition"
            onClick={() => window.open('https://github.com', '_blank')}
          >
            View docs
          </button>
        </div>
        <div className="mt-4 grid gap-3 md:grid-cols-2">
          {runs.map((run) => (
            <article
              key={run.id || run.run_id}
              className="p-4 rounded-xl border border-slate-200 bg-white hover:border-slate-400 transition cursor-pointer"
              onClick={() => onSelectRun(run)}
            >
              <div className="flex items-center justify-between mb-2">
                <p className="text-sm font-semibold text-slate-700">{run.target}</p>
                <span
                  className={`text-xs px-2 py-1 rounded-full ${
                    (run.score || 0) > 80 ? 'bg-emerald-100 text-emerald-700' : 'bg-amber-100 text-amber-700'
                  }`}
                >
                  Score {Math.round(run.score || 0)}
                </span>
              </div>
              <p className="text-sm text-slate-600 line-clamp-2">{run.summary}</p>
              <p className="text-xs text-slate-400 mt-2">{new Date(run.timestamp).toLocaleString()}</p>
            </article>
          ))}
        </div>
        {loading && <p className="text-sm text-slate-500">Loading from backend…</p>}
      </div>
    </section>
  )
}
