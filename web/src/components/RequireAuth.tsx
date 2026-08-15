import { Navigate, useLocation } from 'react-router-dom'
import type { ReactNode } from 'react'

interface RequireAuthProps {
  children: ReactNode
}

// Route guard: redirects to /login when no auth token is present.
export default function RequireAuth({ children }: RequireAuthProps) {
  const location = useLocation()
  const token = localStorage.getItem('auth_token')

  if (!token) {
    return <Navigate to="/login" state={{ from: location.pathname }} replace />
  }

  return <>{children}</>
}
