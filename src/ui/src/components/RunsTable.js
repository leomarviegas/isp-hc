import React, { useState, useEffect } from 'react';

const RunsTable = () => {
  const [runs, setRuns] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // In a real app, you would fetch this data from the API
    const mockData = [
      { run_id: '1', target: '8.8.8.8', score: 15, summary: 'OK', timestamp: new Date().toISOString() },
      { run_id: '2', target: '1.1.1.1', score: 55, summary: 'Intermittent packet loss', timestamp: new Date().toISOString() },
    ];
    setRuns(mockData);
    setLoading(false);
  }, []);

  if (loading) {
    return <div>Loading...</div>;
  }

  return (
    <div className="flex flex-col">
      <div className="-my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
        <div className="py-2 align-middle inline-block min-w-full sm:px-6 lg:px-8">
          <div className="shadow overflow-hidden border-b border-gray-200 sm:rounded-lg">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Target</th>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Score</th>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Summary</th>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Timestamp</th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {runs.map((run) => (
                  <tr key={run.run_id}>
                    <td className="px-6 py-4 whitespace-nowrap">{run.target}</td>
                    <td className="px-6 py-4 whitespace-nowrap">{run.score}</td>
                    <td className="px-6 py-4 whitespace-nowrap">{run.summary}</td>
                    <td className="px-6 py-4 whitespace-nowrap">{run.timestamp}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
};

export default RunsTable;
