/**
 * Main App component with routing
 */
import React from 'react';
import { BrowserRouter as Router, Routes, Route, Link } from 'react-router-dom';
import { RunsProvider } from './context/RunsContext';
import Dashboard from './pages/Dashboard';
import RunDetails from './pages/RunDetails';

const Navigation = () => (
  <nav className="bg-white shadow">
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
      <div className="flex justify-between h-16">
        <div className="flex">
          <div className="flex-shrink-0 flex items-center">
            <Link to="/" className="flex items-center">
              <svg
                className="h-8 w-8 text-indigo-600"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"
                />
              </svg>
              <span className="ml-2 text-xl font-bold text-gray-900">
                ISP Health Checker
              </span>
            </Link>
          </div>
          <div className="hidden sm:ml-6 sm:flex sm:space-x-8">
            <Link
              to="/"
              className="border-indigo-500 text-gray-900 inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium"
            >
              Dashboard
            </Link>
          </div>
        </div>
        <div className="flex items-center">
          <span className="text-sm text-gray-500">
            v1.0.0
          </span>
        </div>
      </div>
    </div>
  </nav>
);

const Layout = ({ children }) => (
  <div className="min-h-screen bg-gray-100">
    <Navigation />
    <main className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
      <div className="px-4 py-6 sm:px-0">
        {children}
      </div>
    </main>
    <footer className="bg-white border-t border-gray-200 mt-auto">
      <div className="max-w-7xl mx-auto py-4 px-4 sm:px-6 lg:px-8">
        <p className="text-center text-sm text-gray-500">
          ISP Health Checker - Open Source Network Diagnostics Tool
        </p>
      </div>
    </footer>
  </div>
);

function App() {
  return (
    <Router>
      <RunsProvider>
        <Layout>
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/runs/:runId" element={<RunDetails />} />
          </Routes>
        </Layout>
      </RunsProvider>
    </Router>
  );
}

export default App;
