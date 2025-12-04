/**
 * Tests for RunsTable component
 */
import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import RunsTable from './RunsTable';

describe('RunsTable', () => {
  const mockRuns = [
    {
      run_id: '1',
      target: '8.8.8.8',
      mode: 'full',
      score: 95,
      summary: 'Connection healthy',
      timestamp: '2024-01-01T12:00:00Z',
    },
    {
      run_id: '2',
      target: '1.1.1.1',
      mode: 'ping',
      score: 65,
      summary: 'Some packet loss',
      timestamp: '2024-01-01T11:00:00Z',
    },
    {
      run_id: '3',
      target: '9.9.9.9',
      mode: 'dns',
      score: 30,
      summary: 'DNS resolution failed',
      timestamp: '2024-01-01T10:00:00Z',
    },
  ];

  const mockOnViewRun = jest.fn();

  beforeEach(() => {
    mockOnViewRun.mockClear();
  });

  test('renders loading state', () => {
    render(<RunsTable runs={[]} loading={true} onViewRun={mockOnViewRun} />);

    // Check for loading spinner (by class or role)
    const spinner = document.querySelector('.animate-spin');
    expect(spinner).toBeInTheDocument();
  });

  test('renders empty state when no runs', () => {
    render(<RunsTable runs={[]} loading={false} onViewRun={mockOnViewRun} />);

    expect(screen.getByText(/no health checks yet/i)).toBeInTheDocument();
  });

  test('renders runs correctly', () => {
    render(<RunsTable runs={mockRuns} loading={false} onViewRun={mockOnViewRun} />);

    // Check targets are displayed
    expect(screen.getByText('8.8.8.8')).toBeInTheDocument();
    expect(screen.getByText('1.1.1.1')).toBeInTheDocument();
    expect(screen.getByText('9.9.9.9')).toBeInTheDocument();

    // Check summaries are displayed
    expect(screen.getByText('Connection healthy')).toBeInTheDocument();
    expect(screen.getByText('Some packet loss')).toBeInTheDocument();
  });

  test('displays correct status badges', () => {
    render(<RunsTable runs={mockRuns} loading={false} onViewRun={mockOnViewRun} />);

    // Healthy (score >= 80)
    expect(screen.getByText('Healthy')).toBeInTheDocument();

    // Warning (score 50-80)
    expect(screen.getByText('Warning')).toBeInTheDocument();

    // Critical (score < 50)
    expect(screen.getByText('Critical')).toBeInTheDocument();
  });

  test('calls onViewRun when row is clicked', () => {
    render(<RunsTable runs={mockRuns} loading={false} onViewRun={mockOnViewRun} />);

    // Click on a row
    const row = screen.getByText('8.8.8.8').closest('tr');
    fireEvent.click(row);

    expect(mockOnViewRun).toHaveBeenCalledWith('1');
  });

  test('calls onViewRun when View button is clicked', () => {
    render(<RunsTable runs={mockRuns} loading={false} onViewRun={mockOnViewRun} />);

    // Click on View button
    const viewButtons = screen.getAllByText('View');
    fireEvent.click(viewButtons[0]);

    expect(mockOnViewRun).toHaveBeenCalledWith('1');
  });

  test('displays score in circular badge', () => {
    render(<RunsTable runs={mockRuns} loading={false} onViewRun={mockOnViewRun} />);

    // Check score values are displayed
    expect(screen.getByText('95')).toBeInTheDocument();
    expect(screen.getByText('65')).toBeInTheDocument();
    expect(screen.getByText('30')).toBeInTheDocument();
  });

  test('displays mode under target', () => {
    render(<RunsTable runs={mockRuns} loading={false} onViewRun={mockOnViewRun} />);

    // Check modes are displayed
    expect(screen.getByText('full')).toBeInTheDocument();
    expect(screen.getByText('ping')).toBeInTheDocument();
    expect(screen.getByText('dns')).toBeInTheDocument();
  });

  test('renders table headers', () => {
    render(<RunsTable runs={mockRuns} loading={false} onViewRun={mockOnViewRun} />);

    expect(screen.getByText('Target')).toBeInTheDocument();
    expect(screen.getByText('Score')).toBeInTheDocument();
    expect(screen.getByText('Status')).toBeInTheDocument();
    expect(screen.getByText('Summary')).toBeInTheDocument();
    expect(screen.getByText('Timestamp')).toBeInTheDocument();
  });
});
