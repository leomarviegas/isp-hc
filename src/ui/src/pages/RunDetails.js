/**
 * Run Details page - displays full details of a health check run
 */
import React, { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useRuns } from '../context/RunsContext';
import { getRunRaw } from '../services/api';
import PacketHealthPanel from '../components/PacketHealthPanel';

const StatusBadge = ({ status }) => {
  const colors = {
    ok: 'bg-green-100 text-green-800',
    OK: 'bg-green-100 text-green-800',
    fail: 'bg-red-100 text-red-800',
    CRIT: 'bg-red-100 text-red-800',
    WARN: 'bg-yellow-100 text-yellow-800',
    na: 'bg-gray-100 text-gray-800',
    NA: 'bg-gray-100 text-gray-800',
  };

  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${colors[status] || 'bg-gray-100 text-gray-800'}`}>
      {status}
    </span>
  );
};

const ScoreMeter = ({ score }) => {
  const getColor = (score) => {
    if (score >= 80) return 'bg-green-500';
    if (score >= 50) return 'bg-yellow-500';
    return 'bg-red-500';
  };

  return (
    <div className="relative pt-1">
      <div className="flex mb-2 items-center justify-between">
        <div>
          <span className={`text-xs font-semibold inline-block py-1 px-2 uppercase rounded-full ${score >= 80 ? 'text-green-600 bg-green-200' : score >= 50 ? 'text-yellow-600 bg-yellow-200' : 'text-red-600 bg-red-200'}`}>
            Health Score
          </span>
        </div>
        <div className="text-right">
          <span className="text-xs font-semibold inline-block text-gray-600">
            {score.toFixed(1)}%
          </span>
        </div>
      </div>
      <div className="overflow-hidden h-2 text-xs flex rounded bg-gray-200">
        <div
          style={{ width: `${score}%` }}
          className={`shadow-none flex flex-col text-center whitespace-nowrap text-white justify-center ${getColor(score)}`}
        />
      </div>
    </div>
  );
};

// Helper to format probe details nicely
const formatProbeDetails = (probe) => {
  if (!probe.details) return null;

  // For probes with raw text output (ping, traceroute), show the text nicely
  if (probe.details.raw && typeof probe.details.raw === 'string') {
    return (
      <pre className="text-xs bg-gray-900 text-green-400 p-3 rounded font-mono whitespace-pre-wrap max-h-64 overflow-y-auto">
        {probe.details.raw}
      </pre>
    );
  }

  // For other probes, show JSON but exclude verbose fields
  const displayDetails = { ...probe.details };
  delete displayDetails.raw; // Already shown above if present

  if (Object.keys(displayDetails).length === 0) return null;

  return (
    <pre className="text-xs bg-gray-50 p-2 rounded overflow-x-auto max-h-48 overflow-y-auto">
      {JSON.stringify(displayDetails, null, 2)}
    </pre>
  );
};

const RunDetails = () => {
  const { runId } = useParams();
  const navigate = useNavigate();
  const { selectedRun, loading, error, fetchRun, deleteRun, clearError } = useRuns();
  const [rawOutput, setRawOutput] = useState(null);
  const [showRaw, setShowRaw] = useState(false);
  const [deleting, setDeleting] = useState(false);

  // Check if run is still pending
  const isPending = selectedRun?.summary?.includes('in progress') ||
                    (selectedRun?.score === 0 && (!selectedRun?.probes || selectedRun?.probes?.length === 0));

  useEffect(() => {
    fetchRun(runId);
  }, [runId, fetchRun]);

  // Auto-refresh when run is pending
  useEffect(() => {
    if (!isPending) return;

    const intervalId = setInterval(() => {
      fetchRun(runId);
    }, 2000); // Poll every 2 seconds

    return () => clearInterval(intervalId);
  }, [isPending, runId, fetchRun]);

  const handleShowRaw = async () => {
    if (!rawOutput) {
      try {
        const data = await getRunRaw(runId);
        setRawOutput(data);
      } catch (err) {
        // Handle error
      }
    }
    setShowRaw(!showRaw);
  };

  const handleDelete = async () => {
    if (window.confirm('Are you sure you want to delete this run?')) {
      setDeleting(true);
      try {
        await deleteRun(runId);
        navigate('/');
      } catch (err) {
        setDeleting(false);
      }
    }
  };

  if (loading && !selectedRun) {
    return (
      <div className="flex justify-center items-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-600"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-md bg-red-50 p-4">
        <div className="flex">
          <div className="ml-3">
            <h3 className="text-sm font-medium text-red-800">Error loading run</h3>
            <p className="mt-2 text-sm text-red-700">{error}</p>
            <button
              onClick={() => navigate('/')}
              className="mt-4 text-sm text-red-600 hover:text-red-500"
            >
              Back to Dashboard
            </button>
          </div>
        </div>
      </div>
    );
  }

  if (!selectedRun) {
    return null;
  }

  const run = selectedRun;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex justify-between items-start">
        <div>
          <button
            onClick={() => navigate('/')}
            className="text-sm text-indigo-600 hover:text-indigo-500 mb-2"
          >
            &larr; Back to Dashboard
          </button>
          <h2 className="text-2xl font-semibold text-gray-900">Run Details</h2>
          <p className="mt-1 text-sm text-gray-500">
            Target: <span className="font-mono">{run.target}</span>
          </p>
        </div>
        <button
          onClick={handleDelete}
          disabled={deleting}
          className="inline-flex items-center px-3 py-2 border border-red-300 text-sm font-medium rounded-md text-red-700 bg-white hover:bg-red-50 disabled:opacity-50"
        >
          {deleting ? 'Deleting...' : 'Delete Run'}
        </button>
      </div>

      {/* Overview Card */}
      <div className="bg-white shadow rounded-lg p-6">
        <h3 className="text-lg font-medium text-gray-900 mb-4">Overview</h3>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          <div>
            <dt className="text-sm font-medium text-gray-500">Run ID</dt>
            <dd className="mt-1 text-sm text-gray-900 font-mono">{run.run_id}</dd>
          </div>
          <div>
            <dt className="text-sm font-medium text-gray-500">Timestamp</dt>
            <dd className="mt-1 text-sm text-gray-900">
              {new Date(run.timestamp).toLocaleString()}
            </dd>
          </div>
          <div>
            <dt className="text-sm font-medium text-gray-500">Mode</dt>
            <dd className="mt-1 text-sm text-gray-900 capitalize">{run.mode}</dd>
          </div>
          <div>
            <dt className="text-sm font-medium text-gray-500">Summary</dt>
            <dd className="mt-1 text-sm text-gray-900 flex items-center gap-2">
              {run.summary}
              {isPending && (
                <span className="inline-flex items-center">
                  <svg className="animate-spin h-4 w-4 text-indigo-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                </span>
              )}
            </dd>
          </div>
        </div>
        <div className="mt-6">
          <ScoreMeter score={run.score} />
        </div>
      </div>

      {/* Probes Card */}
      <div className="bg-white shadow rounded-lg p-6">
        <h3 className="text-lg font-medium text-gray-900 mb-4">Probe Results</h3>
        {run.probes && run.probes.length > 0 ? (
          <div className="overflow-hidden">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Probe</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Latency</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Details</th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {run.probes.map((probe, index) => (
                  <tr key={index}>
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 capitalize">
                      {probe.name}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <StatusBadge status={probe.status} />
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {probe.latency_ms ? `${probe.latency_ms.toFixed(2)} ms` : '-'}
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-500 max-w-xl">
                      {formatProbeDetails(probe)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : isPending ? (
          <div className="flex items-center gap-3 text-sm text-indigo-600">
            <svg className="animate-spin h-5 w-5" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            Running health check probes...
          </div>
        ) : (
          <p className="text-sm text-gray-500">No probe results available.</p>
        )}
      </div>

      {/* Packet Health Panel - displays interface stats, TCP stats, and packet capture results */}
      <PacketHealthPanel probes={run.probes} />

      {/* Diagnosis Card */}
      {run.diagnosis && run.diagnosis.length > 0 && (
        <div className="bg-white shadow rounded-lg p-6">
          <h3 className="text-lg font-medium text-gray-900 mb-4">Diagnosis</h3>
          <div className="space-y-4">
            {run.diagnosis.map((diag, index) => (
              <div key={index} className="border-l-4 border-indigo-400 pl-4">
                <div className="flex items-center justify-between">
                  <h4 className="text-sm font-medium text-gray-900">{diag.component}</h4>
                  <span className="text-xs text-gray-500">
                    Confidence: {(diag.confidence * 100).toFixed(0)}%
                  </span>
                </div>
                <p className="mt-1 text-sm text-gray-600">{diag.explanation}</p>
                {diag.suggested_action && (
                  <p className="mt-2 text-sm text-indigo-600">
                    Suggested Action: {diag.suggested_action}
                  </p>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Raw Output Card */}
      <div className="bg-white shadow rounded-lg p-6">
        <div className="flex justify-between items-center mb-4">
          <h3 className="text-lg font-medium text-gray-900">Raw Output</h3>
          <button
            onClick={handleShowRaw}
            className="text-sm text-indigo-600 hover:text-indigo-500"
          >
            {showRaw ? 'Hide' : 'Show'} Raw Output
          </button>
        </div>
        {showRaw && rawOutput && (
          <pre className="bg-gray-900 text-gray-100 p-4 rounded-lg overflow-x-auto text-xs">
            {JSON.stringify(rawOutput, null, 2)}
          </pre>
        )}
      </div>
    </div>
  );
};

export default RunDetails;
