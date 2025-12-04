/**
 * Dashboard page - displays list of runs and quick actions
 */
import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useRuns } from '../context/RunsContext';
import RunsTable from '../components/RunsTable';
import NewRunForm from '../components/NewRunForm';

const Dashboard = () => {
  const navigate = useNavigate();
  const { runs, loading, error, fetchRuns, createRun, clearError } = useRuns();
  const [showNewRunForm, setShowNewRunForm] = useState(false);

  useEffect(() => {
    fetchRuns(true);
  }, [fetchRuns]);

  const handleCreateRun = async (runData) => {
    try {
      const response = await createRun(runData);
      setShowNewRunForm(false);
      // Navigate to the new run's detail page
      if (response.run_id) {
        navigate(`/runs/${response.run_id}`);
      }
    } catch (err) {
      // Error is handled by context
    }
  };

  const handleViewRun = (runId) => {
    navigate(`/runs/${runId}`);
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-2xl font-semibold text-gray-900">Health Check Runs</h2>
          <p className="mt-1 text-sm text-gray-500">
            View and manage your ISP health check diagnostics
          </p>
        </div>
        <button
          onClick={() => setShowNewRunForm(true)}
          className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
        >
          <svg className="-ml-1 mr-2 h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          New Health Check
        </button>
      </div>

      {/* Error display */}
      {error && (
        <div className="rounded-md bg-red-50 p-4">
          <div className="flex">
            <div className="flex-shrink-0">
              <svg className="h-5 w-5 text-red-400" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
              </svg>
            </div>
            <div className="ml-3">
              <p className="text-sm font-medium text-red-800">{error}</p>
            </div>
            <div className="ml-auto pl-3">
              <button
                onClick={clearError}
                className="inline-flex rounded-md bg-red-50 p-1.5 text-red-500 hover:bg-red-100"
              >
                <svg className="h-5 w-5" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
                </svg>
              </button>
            </div>
          </div>
        </div>
      )}

      {/* New Run Modal */}
      {showNewRunForm && (
        <NewRunForm
          onSubmit={handleCreateRun}
          onCancel={() => setShowNewRunForm(false)}
          loading={loading}
        />
      )}

      {/* Runs Table */}
      <RunsTable
        runs={runs}
        loading={loading}
        onViewRun={handleViewRun}
      />
    </div>
  );
};

export default Dashboard;
