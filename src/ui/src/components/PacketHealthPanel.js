/**
 * PacketHealthPanel - Displays packet-level health metrics
 * Shows interface stats, TCP stats, and packet capture results
 */
import React from 'react';

const PacketHealthPanel = ({ probes }) => {
  // Filter packet health probes
  const interfaceStats = probes?.find(p => p.name === 'interface_stats');
  const tcpStats = probes?.find(p => p.name === 'tcp_stats');
  const packetCapture = probes?.find(p => p.name === 'packet_capture');
  const socketStats = probes?.find(p => p.name === 'socket_stats');

  const hasPacketProbes = interfaceStats || tcpStats || packetCapture || socketStats;

  if (!hasPacketProbes) {
    return null;
  }

  return (
    <div className="bg-white rounded-lg shadow p-6 mt-6">
      <h3 className="text-lg font-semibold text-gray-900 mb-4 flex items-center">
        <svg className="w-5 h-5 mr-2 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
        </svg>
        Packet Health Analysis
      </h3>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Interface Stats */}
        {interfaceStats && (
          <InterfaceStatsCard probe={interfaceStats} />
        )}

        {/* TCP Stats */}
        {tcpStats && (
          <TCPStatsCard probe={tcpStats} />
        )}

        {/* Packet Capture */}
        {packetCapture && (
          <PacketCaptureCard probe={packetCapture} />
        )}

        {/* Socket Stats */}
        {socketStats && (
          <SocketStatsCard probe={socketStats} />
        )}
      </div>
    </div>
  );
};

const StatusBadge = ({ status }) => {
  const colors = {
    ok: 'bg-green-100 text-green-800',
    warn: 'bg-yellow-100 text-yellow-800',
    fail: 'bg-red-100 text-red-800',
    na: 'bg-gray-100 text-gray-600',
  };

  return (
    <span className={`px-2 py-1 text-xs font-medium rounded-full ${colors[status] || colors.na}`}>
      {status?.toUpperCase()}
    </span>
  );
};

const MetricRow = ({ label, value, unit = '', warning = false, critical = false }) => {
  let valueColor = 'text-gray-900';
  if (critical) valueColor = 'text-red-600 font-semibold';
  else if (warning) valueColor = 'text-yellow-600';

  return (
    <div className="flex justify-between items-center py-1">
      <span className="text-sm text-gray-600">{label}</span>
      <span className={`text-sm ${valueColor}`}>
        {typeof value === 'number' ? value.toLocaleString() : value}{unit}
      </span>
    </div>
  );
};

const InterfaceStatsCard = ({ probe }) => {
  const details = probe.details || {};
  const errorRate = details.error_rate_percent || 0;
  const dropRate = details.drop_rate_percent || 0;

  return (
    <div className="border rounded-lg p-4">
      <div className="flex justify-between items-center mb-3">
        <h4 className="font-medium text-gray-900">Interface Statistics</h4>
        <StatusBadge status={probe.status} />
      </div>

      <div className="space-y-1">
        <MetricRow
          label="Total Packets"
          value={details.total_packets}
        />
        <MetricRow
          label="Errors"
          value={details.total_errors}
          warning={details.total_errors > 0}
          critical={errorRate > 1}
        />
        <MetricRow
          label="Error Rate"
          value={errorRate.toFixed(3)}
          unit="%"
          warning={errorRate > 0.1}
          critical={errorRate > 1}
        />
        <MetricRow
          label="Dropped"
          value={details.total_dropped}
          warning={details.total_dropped > 0}
          critical={dropRate > 1}
        />
        <MetricRow
          label="Drop Rate"
          value={dropRate.toFixed(3)}
          unit="%"
          warning={dropRate > 0.1}
          critical={dropRate > 1}
        />
      </div>

      {details.problem_interfaces?.length > 0 && (
        <div className="mt-3 p-2 bg-yellow-50 rounded text-sm">
          <span className="text-yellow-800 font-medium">Problem interfaces: </span>
          <span className="text-yellow-700">{details.problem_interfaces.join(', ')}</span>
        </div>
      )}
    </div>
  );
};

const TCPStatsCard = ({ probe }) => {
  const details = probe.details || {};
  const stats = details.stats || {};
  const retransRate = details.retransmission_rate || 0;
  const ofoRate = details.out_of_order_rate || 0;

  return (
    <div className="border rounded-lg p-4">
      <div className="flex justify-between items-center mb-3">
        <h4 className="font-medium text-gray-900">TCP Statistics</h4>
        <StatusBadge status={probe.status} />
      </div>

      <div className="space-y-1">
        <MetricRow
          label="Active Connections"
          value={stats.curr_estab || details.current_connections}
        />
        <MetricRow
          label="Segments In"
          value={stats.in_segs}
        />
        <MetricRow
          label="Segments Out"
          value={stats.out_segs}
        />
        <MetricRow
          label="Retransmissions"
          value={stats.retrans_segs}
          warning={retransRate > 1}
          critical={retransRate > 5}
        />
        <MetricRow
          label="Retrans Rate"
          value={retransRate.toFixed(2)}
          unit="%"
          warning={retransRate > 1}
          critical={retransRate > 5}
        />
        <MetricRow
          label="Reorder Events"
          value={details.total_reorder_events}
          warning={details.total_reorder_events > 100}
        />
        <MetricRow
          label="Out-of-Order Rate"
          value={ofoRate.toFixed(2)}
          unit="%"
          warning={ofoRate > 0.5}
        />
        <MetricRow
          label="Errors In"
          value={stats.in_errs}
          critical={stats.in_errs > 0}
        />
      </div>

      {details.issues?.length > 0 && (
        <div className="mt-3 p-2 bg-red-50 rounded">
          <ul className="text-sm text-red-700 space-y-1">
            {details.issues.map((issue, i) => (
              <li key={i}>• {issue}</li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
};

const PacketCaptureCard = ({ probe }) => {
  const details = probe.details || {};
  const stats = details.stats || {};

  if (probe.status === 'na') {
    return (
      <div className="border rounded-lg p-4">
        <div className="flex justify-between items-center mb-3">
          <h4 className="font-medium text-gray-900">Packet Capture</h4>
          <StatusBadge status={probe.status} />
        </div>
        <p className="text-sm text-gray-500">{probe.error || 'Not available'}</p>
      </div>
    );
  }

  return (
    <div className="border rounded-lg p-4">
      <div className="flex justify-between items-center mb-3">
        <h4 className="font-medium text-gray-900">Packet Capture Analysis</h4>
        <StatusBadge status={probe.status} />
      </div>

      <div className="space-y-1">
        <MetricRow label="Packets Captured" value={stats.packet_count} />
        <MetricRow label="TCP Packets" value={stats.tcp_packets} />
        <MetricRow
          label="Retransmits"
          value={stats.tcp_retransmits}
          warning={stats.tcp_retransmits > 10}
          critical={stats.tcp_retransmits > 50}
        />
        <MetricRow
          label="Out-of-Order"
          value={stats.tcp_out_of_order}
          warning={stats.tcp_out_of_order > 10}
        />
        <MetricRow
          label="Duplicate ACKs"
          value={stats.tcp_duplicate_acks}
          warning={stats.tcp_duplicate_acks > 20}
        />
        <MetricRow
          label="Zero Window"
          value={stats.tcp_zero_window}
          warning={stats.tcp_zero_window > 0}
        />
        <MetricRow
          label="RST Packets"
          value={stats.tcp_rst_packets}
          warning={stats.tcp_rst_packets > 10}
        />
      </div>

      {/* Critical Issues Section */}
      {(stats.checksum_errors > 0 || stats.malformed_packets > 0) && (
        <div className="mt-3 p-3 bg-red-100 border border-red-300 rounded">
          <h5 className="text-red-800 font-semibold text-sm mb-2">⚠️ Packet Corruption Detected</h5>
          <div className="space-y-1">
            {stats.checksum_errors > 0 && (
              <MetricRow
                label="Checksum Errors"
                value={stats.checksum_errors}
                critical={true}
              />
            )}
            {stats.malformed_packets > 0 && (
              <MetricRow
                label="Malformed Packets"
                value={stats.malformed_packets}
                critical={true}
              />
            )}
          </div>
        </div>
      )}

      {details.issues?.length > 0 && (
        <div className="mt-3 p-2 bg-yellow-50 rounded">
          <ul className="text-sm text-yellow-700 space-y-1">
            {details.issues.map((issue, i) => (
              <li key={i}>• {issue}</li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
};

const SocketStatsCard = ({ probe }) => {
  const details = probe.details || {};
  const sockets = details.sockets || [];

  if (probe.status === 'na') {
    return (
      <div className="border rounded-lg p-4">
        <div className="flex justify-between items-center mb-3">
          <h4 className="font-medium text-gray-900">Socket Statistics</h4>
          <StatusBadge status={probe.status} />
        </div>
        <p className="text-sm text-gray-500">{probe.error || 'Not available'}</p>
      </div>
    );
  }

  return (
    <div className="border rounded-lg p-4">
      <div className="flex justify-between items-center mb-3">
        <h4 className="font-medium text-gray-900">Active Sockets</h4>
        <StatusBadge status={probe.status} />
      </div>

      {sockets.length === 0 ? (
        <p className="text-sm text-gray-500">No active connections</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="min-w-full text-sm">
            <thead>
              <tr className="text-left text-gray-600">
                <th className="pb-2">Remote</th>
                <th className="pb-2">RTT</th>
                <th className="pb-2">Retrans</th>
                <th className="pb-2">State</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {sockets.slice(0, 5).map((socket, i) => (
                <tr key={i}>
                  <td className="py-1 font-mono text-xs">{socket.remote}</td>
                  <td className={`py-1 ${socket.rtt_ms > 200 ? 'text-yellow-600' : ''}`}>
                    {socket.rtt_ms?.toFixed(1)}ms
                  </td>
                  <td className={`py-1 ${socket.retransmits > 5 ? 'text-red-600' : ''}`}>
                    {socket.retransmits}
                  </td>
                  <td className="py-1">{socket.state}</td>
                </tr>
              ))}
            </tbody>
          </table>
          {sockets.length > 5 && (
            <p className="text-xs text-gray-500 mt-2">
              +{sockets.length - 5} more connections
            </p>
          )}
        </div>
      )}

      {details.issues?.length > 0 && (
        <div className="mt-3 p-2 bg-yellow-50 rounded">
          <ul className="text-sm text-yellow-700 space-y-1">
            {details.issues.map((issue, i) => (
              <li key={i}>• {issue}</li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
};

export default PacketHealthPanel;
