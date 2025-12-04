/**
 * New Run Form component - modal for creating a new health check
 */
import React, { useState } from 'react';

const modeDescriptions = {
  full: 'Runs ping, DNS resolution, and traceroute to check basic network connectivity.',
  ping: 'Tests if the target host is reachable and measures round-trip latency.',
  dns: 'Checks DNS resolution for the target domain.',
  traceroute: 'Maps the network path to the target, showing each hop.',
  packet: 'Comprehensive packet health analysis: interface errors, TCP stats, socket state, and packet capture.',
  interface: 'Reads network interface counters for CRC errors, frame errors, drops, and overruns.',
  tcp: 'Analyzes TCP protocol statistics: retransmissions, reordering, duplicate ACKs, out-of-order packets.',
  socket: 'Shows active TCP socket states and per-connection retransmit counts.',
  capture: 'Performs deep packet inspection using tcpdump to detect malformed packets and checksum errors.',
  comprehensive: 'Runs all available probes for a complete network health assessment.',
};

const NewRunForm = ({ onSubmit, onCancel, loading }) => {
  const [target, setTarget] = useState('8.8.8.8');
  const [mode, setMode] = useState('full');

  const handleSubmit = (e) => {
    e.preventDefault();
    onSubmit({
      target,
      mode,
      score: 0,
      summary: 'Pending...',
    });
  };

  return (
    <div className="fixed inset-0 z-10 overflow-y-auto">
      <div className="flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0">
        <div className="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity" onClick={onCancel} />

        <div className="relative transform overflow-hidden rounded-lg bg-white text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-lg">
          <form onSubmit={handleSubmit}>
            <div className="bg-white px-4 pb-4 pt-5 sm:p-6 sm:pb-4">
              <div className="sm:flex sm:items-start">
                <div className="mx-auto flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full bg-indigo-100 sm:mx-0 sm:h-10 sm:w-10">
                  <svg className="h-6 w-6 text-indigo-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                </div>
                <div className="mt-3 text-center sm:ml-4 sm:mt-0 sm:text-left flex-1">
                  <h3 className="text-lg font-semibold leading-6 text-gray-900">
                    New Health Check
                  </h3>
                  <div className="mt-4 space-y-4">
                    <div>
                      <label htmlFor="target" className="block text-sm font-medium text-gray-700">
                        Target Host/IP
                      </label>
                      <input
                        type="text"
                        id="target"
                        value={target}
                        onChange={(e) => setTarget(e.target.value)}
                        className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
                        placeholder="8.8.8.8 or example.com"
                        required
                      />
                    </div>
                    <div>
                      <label htmlFor="mode" className="block text-sm font-medium text-gray-700">
                        Check Mode
                      </label>
                      <select
                        id="mode"
                        value={mode}
                        onChange={(e) => setMode(e.target.value)}
                        className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
                      >
                        <optgroup label="Basic Connectivity">
                          <option value="full">Full (Ping + DNS + Traceroute)</option>
                          <option value="ping">Ping only</option>
                          <option value="dns">DNS only</option>
                          <option value="traceroute">Traceroute only</option>
                        </optgroup>
                        <optgroup label="Packet Health Analysis">
                          <option value="packet">Packet Health (All packet probes)</option>
                          <option value="interface">Interface Stats (errors, drops)</option>
                          <option value="tcp">TCP Stats (retransmits, reordering)</option>
                          <option value="socket">Socket Stats (connection details)</option>
                          <option value="capture">Packet Capture (deep inspection)</option>
                        </optgroup>
                        <optgroup label="Comprehensive">
                          <option value="comprehensive">All Probes (connectivity + packet health)</option>
                        </optgroup>
                      </select>
                      <p className="mt-2 text-xs text-gray-500">
                        {modeDescriptions[mode]}
                      </p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            <div className="bg-gray-50 px-4 py-3 sm:flex sm:flex-row-reverse sm:px-6">
              <button
                type="submit"
                disabled={loading}
                className="inline-flex w-full justify-center rounded-md bg-indigo-600 px-3 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 sm:ml-3 sm:w-auto disabled:opacity-50"
              >
                {loading ? 'Starting...' : 'Start Health Check'}
              </button>
              <button
                type="button"
                onClick={onCancel}
                disabled={loading}
                className="mt-3 inline-flex w-full justify-center rounded-md bg-white px-3 py-2 text-sm font-semibold text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 hover:bg-gray-50 sm:mt-0 sm:w-auto"
              >
                Cancel
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
};

export default NewRunForm;
