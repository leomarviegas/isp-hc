/**
 * Tests for NewRunForm component
 */
import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import NewRunForm from './NewRunForm';

describe('NewRunForm', () => {
  const mockOnSubmit = jest.fn();
  const mockOnCancel = jest.fn();

  beforeEach(() => {
    mockOnSubmit.mockClear();
    mockOnCancel.mockClear();
  });

  test('renders form fields', () => {
    render(
      <NewRunForm
        onSubmit={mockOnSubmit}
        onCancel={mockOnCancel}
        loading={false}
      />
    );

    expect(screen.getByLabelText(/target host/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/check mode/i)).toBeInTheDocument();
  });

  test('has default target value', () => {
    render(
      <NewRunForm
        onSubmit={mockOnSubmit}
        onCancel={mockOnCancel}
        loading={false}
      />
    );

    const targetInput = screen.getByLabelText(/target host/i);
    expect(targetInput.value).toBe('8.8.8.8');
  });

  test('has default mode value', () => {
    render(
      <NewRunForm
        onSubmit={mockOnSubmit}
        onCancel={mockOnCancel}
        loading={false}
      />
    );

    const modeSelect = screen.getByLabelText(/check mode/i);
    expect(modeSelect.value).toBe('full');
  });

  test('calls onSubmit with correct data', () => {
    render(
      <NewRunForm
        onSubmit={mockOnSubmit}
        onCancel={mockOnCancel}
        loading={false}
      />
    );

    // Change target
    const targetInput = screen.getByLabelText(/target host/i);
    fireEvent.change(targetInput, { target: { value: '1.1.1.1' } });

    // Change mode
    const modeSelect = screen.getByLabelText(/check mode/i);
    fireEvent.change(modeSelect, { target: { value: 'ping' } });

    // Submit form
    const submitButton = screen.getByText(/start health check/i);
    fireEvent.click(submitButton);

    expect(mockOnSubmit).toHaveBeenCalledWith({
      target: '1.1.1.1',
      mode: 'ping',
      score: 0,
      summary: 'Pending...',
    });
  });

  test('calls onCancel when cancel button is clicked', () => {
    render(
      <NewRunForm
        onSubmit={mockOnSubmit}
        onCancel={mockOnCancel}
        loading={false}
      />
    );

    const cancelButton = screen.getByText(/cancel/i);
    fireEvent.click(cancelButton);

    expect(mockOnCancel).toHaveBeenCalled();
  });

  test('disables buttons when loading', () => {
    render(
      <NewRunForm
        onSubmit={mockOnSubmit}
        onCancel={mockOnCancel}
        loading={true}
      />
    );

    const submitButton = screen.getByText(/starting/i);
    const cancelButton = screen.getByText(/cancel/i);

    expect(submitButton).toBeDisabled();
    expect(cancelButton).toBeDisabled();
  });

  test('shows loading text when submitting', () => {
    render(
      <NewRunForm
        onSubmit={mockOnSubmit}
        onCancel={mockOnCancel}
        loading={true}
      />
    );

    expect(screen.getByText(/starting/i)).toBeInTheDocument();
  });

  test('renders mode options', () => {
    render(
      <NewRunForm
        onSubmit={mockOnSubmit}
        onCancel={mockOnCancel}
        loading={false}
      />
    );

    expect(screen.getByText(/full \(all probes\)/i)).toBeInTheDocument();
    expect(screen.getByText(/ping only/i)).toBeInTheDocument();
    expect(screen.getByText(/dns only/i)).toBeInTheDocument();
    expect(screen.getByText(/traceroute only/i)).toBeInTheDocument();
  });
});
