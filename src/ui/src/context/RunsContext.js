/**
 * React Context for managing runs state
 * Provides centralized state management for the application
 */
import React, { createContext, useContext, useReducer, useCallback } from 'react';
import * as api from '../services/api';

// Initial state
const initialState = {
  runs: [],
  selectedRun: null,
  loading: false,
  error: null,
  pagination: {
    limit: 20,
    offset: 0,
    hasMore: true,
  },
  filters: {
    target: '',
  },
};

// Action types
const ActionTypes = {
  SET_LOADING: 'SET_LOADING',
  SET_ERROR: 'SET_ERROR',
  SET_RUNS: 'SET_RUNS',
  APPEND_RUNS: 'APPEND_RUNS',
  SET_SELECTED_RUN: 'SET_SELECTED_RUN',
  ADD_RUN: 'ADD_RUN',
  REMOVE_RUN: 'REMOVE_RUN',
  SET_FILTERS: 'SET_FILTERS',
  RESET_PAGINATION: 'RESET_PAGINATION',
  CLEAR_ERROR: 'CLEAR_ERROR',
};

// Reducer function
const runsReducer = (state, action) => {
  switch (action.type) {
    case ActionTypes.SET_LOADING:
      return { ...state, loading: action.payload };

    case ActionTypes.SET_ERROR:
      return { ...state, error: action.payload, loading: false };

    case ActionTypes.CLEAR_ERROR:
      return { ...state, error: null };

    case ActionTypes.SET_RUNS:
      return {
        ...state,
        runs: action.payload,
        loading: false,
        error: null,
        pagination: {
          ...state.pagination,
          offset: action.payload.length,
          hasMore: action.payload.length >= state.pagination.limit,
        },
      };

    case ActionTypes.APPEND_RUNS:
      return {
        ...state,
        runs: [...state.runs, ...action.payload],
        loading: false,
        pagination: {
          ...state.pagination,
          offset: state.pagination.offset + action.payload.length,
          hasMore: action.payload.length >= state.pagination.limit,
        },
      };

    case ActionTypes.SET_SELECTED_RUN:
      return { ...state, selectedRun: action.payload, loading: false };

    case ActionTypes.ADD_RUN:
      return {
        ...state,
        runs: [action.payload, ...state.runs],
      };

    case ActionTypes.REMOVE_RUN:
      return {
        ...state,
        runs: state.runs.filter((run) => run.run_id !== action.payload),
        selectedRun: state.selectedRun?.run_id === action.payload ? null : state.selectedRun,
      };

    case ActionTypes.SET_FILTERS:
      return {
        ...state,
        filters: { ...state.filters, ...action.payload },
      };

    case ActionTypes.RESET_PAGINATION:
      return {
        ...state,
        pagination: { ...initialState.pagination },
        runs: [],
      };

    default:
      return state;
  }
};

// Create context
const RunsContext = createContext(null);

// Provider component
export const RunsProvider = ({ children }) => {
  const [state, dispatch] = useReducer(runsReducer, initialState);

  // Fetch runs
  const fetchRuns = useCallback(async (reset = false) => {
    dispatch({ type: ActionTypes.SET_LOADING, payload: true });

    try {
      const offset = reset ? 0 : state.pagination.offset;
      const data = await api.getRuns({
        limit: state.pagination.limit,
        offset,
        target: state.filters.target || undefined,
      });

      if (reset) {
        dispatch({ type: ActionTypes.SET_RUNS, payload: data });
      } else {
        dispatch({ type: ActionTypes.APPEND_RUNS, payload: data });
      }
    } catch (error) {
      dispatch({ type: ActionTypes.SET_ERROR, payload: error.message });
    }
  }, [state.pagination.limit, state.pagination.offset, state.filters.target]);

  // Fetch a single run
  const fetchRun = useCallback(async (runId) => {
    dispatch({ type: ActionTypes.SET_LOADING, payload: true });

    try {
      const data = await api.getRun(runId);
      dispatch({ type: ActionTypes.SET_SELECTED_RUN, payload: data });
      return data;
    } catch (error) {
      dispatch({ type: ActionTypes.SET_ERROR, payload: error.message });
      throw error;
    }
  }, []);

  // Create a new run
  const createRun = useCallback(async (runData) => {
    dispatch({ type: ActionTypes.SET_LOADING, payload: true });

    try {
      const response = await api.createRun(runData);
      // The run will be added when we fetch it after creation
      dispatch({ type: ActionTypes.SET_LOADING, payload: false });
      return response;
    } catch (error) {
      dispatch({ type: ActionTypes.SET_ERROR, payload: error.message });
      throw error;
    }
  }, []);

  // Delete a run
  const deleteRun = useCallback(async (runId) => {
    try {
      await api.deleteRun(runId);
      dispatch({ type: ActionTypes.REMOVE_RUN, payload: runId });
    } catch (error) {
      dispatch({ type: ActionTypes.SET_ERROR, payload: error.message });
      throw error;
    }
  }, []);

  // Set filters
  const setFilters = useCallback((filters) => {
    dispatch({ type: ActionTypes.SET_FILTERS, payload: filters });
    dispatch({ type: ActionTypes.RESET_PAGINATION });
  }, []);

  // Clear error
  const clearError = useCallback(() => {
    dispatch({ type: ActionTypes.CLEAR_ERROR });
  }, []);

  // Load more runs
  const loadMore = useCallback(() => {
    if (state.pagination.hasMore && !state.loading) {
      fetchRuns(false);
    }
  }, [state.pagination.hasMore, state.loading, fetchRuns]);

  const value = {
    ...state,
    fetchRuns,
    fetchRun,
    createRun,
    deleteRun,
    setFilters,
    clearError,
    loadMore,
  };

  return <RunsContext.Provider value={value}>{children}</RunsContext.Provider>;
};

// Custom hook to use the context
export const useRuns = () => {
  const context = useContext(RunsContext);
  if (!context) {
    throw new Error('useRuns must be used within a RunsProvider');
  }
  return context;
};

export default RunsContext;
