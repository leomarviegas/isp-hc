import Home from './pages/Home'
import RunDetail from './pages/RunDetail'
import { useState } from 'react'

export default function App() {
  const [selectedRun, setSelectedRun] = useState(null)

  return (
    <div className="max-w-5xl mx-auto px-6 py-10 space-y-8">
      <header className="flex items-center justify-between">
        <div>
          <p className="text-sm uppercase tracking-[0.2em] text-slate-500">ISP Health Checker</p>
          <h1 className="text-3xl font-semibold">Visibility for every hop</h1>
        </div>
        <div className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-slate-900 text-white shadow-lg">
          <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
          <span className="text-sm">Demo mode</span>
        </div>
      </header>
      <Home onSelectRun={setSelectedRun} />
      {selectedRun && <RunDetail run={selectedRun} onClose={() => setSelectedRun(null)} />}
    </div>
  )
}
