"use client";

import React from "react";

interface ErrorBoundaryProps {
  children: React.ReactNode;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends React.Component<
  ErrorBoundaryProps,
  ErrorBoundaryState
> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo): void {
    console.error("ErrorBoundary caught an error:", error, errorInfo.componentStack);
  }

  handleReload = () => {
    window.location.reload();
  };

  handleReport = () => {
    const errorDetails = {
      message: this.state.error?.message ?? "Unknown error",
      stack: this.state.error?.stack ?? "N/A",
      timestamp: new Date().toISOString(),
      url: window.location.href,
    };
    // Log the full report to console for collection by monitoring tools
    console.error("Error Report:", JSON.stringify(errorDetails, null, 2));
    // Copy error info to clipboard for manual reporting
    navigator.clipboard?.writeText(JSON.stringify(errorDetails, null, 2)).then(
      () => window.alert("Error details copied to clipboard."),
      () => window.alert("Error details logged to console.")
    );
  };

  render() {
    if (this.state.hasError) {
      return (
        <div className="fixed inset-0 flex items-center justify-center bg-zinc-950 z-50">
          <div className="max-w-md rounded-2xl border border-red-500/30 bg-zinc-900/90 px-8 py-8 text-center shadow-2xl shadow-black/40">
            <div className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-red-500/10 border border-red-500/30">
              <svg
                width="24"
                height="24"
                viewBox="0 0 24 24"
                fill="none"
                stroke="#ef4444"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <circle cx="12" cy="12" r="10" />
                <line x1="12" y1="8" x2="12" y2="12" />
                <line x1="12" y1="16" x2="12.01" y2="16" />
              </svg>
            </div>
            <h2 className="text-lg font-semibold text-zinc-100 mb-2">
              Something went wrong
            </h2>
            <p className="text-sm text-zinc-400 mb-1">
              An unexpected error occurred while rendering the application.
            </p>
            <p className="text-xs text-red-400/80 mb-6 font-mono break-all">
              {this.state.error?.message ?? "Unknown error"}
            </p>
            <div className="flex items-center justify-center gap-3">
              <button
                onClick={this.handleReload}
                className="rounded-lg bg-blue-600 hover:bg-blue-500 px-5 py-2 text-sm font-medium text-white transition-colors"
              >
                Reload
              </button>
              <a
                href="#report"
                onClick={(e) => {
                  e.preventDefault();
                  this.handleReport();
                }}
                className="rounded-lg border border-zinc-700 hover:border-zinc-600 px-5 py-2 text-sm font-medium text-zinc-300 hover:text-zinc-100 transition-colors"
              >
                Report
              </a>
            </div>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
