import { Component, type ErrorInfo, type ReactNode } from 'react'
import { Button } from '@/components/ui/button'

type Props = { children: ReactNode; fallbackTitle?: string }
type State = { error: Error | null }

export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null }

  static getDerivedStateFromError(error: Error): State {
    return { error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('Route error:', error, info)
  }

  render() {
    if (this.state.error) {
      return (
        <div className="flex flex-col items-center justify-center gap-4 rounded-lg border border-[hsl(var(--destructive)/0.4)] bg-[hsl(var(--destructive)/0.08)] px-6 py-16 text-center">
          <h2 className="text-lg font-medium">{this.props.fallbackTitle ?? 'Something went wrong'}</h2>
          <p className="max-w-md text-sm text-[hsl(var(--muted-foreground))]">{this.state.error.message}</p>
          <Button variant="outline" onClick={() => this.setState({ error: null })}>
            Try again
          </Button>
        </div>
      )
    }
    return this.props.children
  }
}
