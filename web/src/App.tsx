import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import Layout from './components/Layout'
import RequireAuth from './components/RequireAuth'
import ConnectionMonitor from './components/ConnectionMonitor'
import VersionCheck from './components/VersionCheck'
import Dashboard from './pages/Dashboard'
import Platforms from './pages/Platforms'
import Adapters from './pages/Adapters'
import Instances from './pages/Instances'
import Messages from './pages/Messages'
import RoutesPage from './pages/Routes'
import Ops from './pages/Ops'
import Settings from './pages/Settings'
import Login from './pages/Login'
import Register from './pages/Register'
import Docs from './pages/Docs'
import DocViewer from './pages/DocViewer'
import AdapterGateway from './pages/AdapterGateway'

function App() {
  return (
    <BrowserRouter>
      {/* Show a modal warning whenever the frontend loses connection to the backend */}
      <ConnectionMonitor />
      {/* Show a modal warning when frontend version doesn't match backend version */}
      <VersionCheck />
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route
          path="/"
          element={
            <RequireAuth>
              <Layout />
            </RequireAuth>
          }
        >
          <Route index element={<Navigate to="/dashboard" replace />} />
          <Route path="dashboard" element={<Dashboard />} />
          <Route path="platforms" element={<Platforms />} />
          <Route path="adapters" element={<Adapters />} />
          <Route path="instances" element={<Instances />} />
          <Route path="messages" element={<Messages />} />
          <Route path="adapter-gateway" element={<AdapterGateway />} />
          <Route path="routes" element={<RoutesPage />} />
          <Route path="ops" element={<Ops />} />
          {/* Legacy path. The app is a `BrowserRouter` SPA served by a
              fallback proxy, so this `<Navigate replace>` *is* the redirect —
              there is no server-side 301 to add. The catch-all-free route
              structure is unchanged. */}
          <Route path="monitoring" element={<Navigate to="/ops" replace />} />
          <Route path="settings" element={<Settings />} />
          <Route path="docs" element={<Docs />} />
          <Route path="docs/:key" element={<DocViewer />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}

export default App
