import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import Layout from './components/Layout'
import Dashboard from './pages/Dashboard'
import Platforms from './pages/Platforms'
import Adapters from './pages/Adapters'
import Instances from './pages/Instances'
import Messages from './pages/Messages'
import RoutesPage from './pages/Routes'
import Monitoring from './pages/Monitoring'
import Settings from './pages/Settings'
import Login from './pages/Login'
import Docs from './pages/Docs'

function App() {
  const { t } = useTranslation()
  
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/" element={<Layout />}>
          <Route index element={<Navigate to="/dashboard" replace />} />
          <Route path="dashboard" element={<Dashboard />} />
          <Route path="platforms" element={<Platforms />} />
          <Route path="adapters" element={<Adapters />} />
          <Route path="instances" element={<Instances />} />
          <Route path="messages" element={<Messages />} />
          <Route path="routes" element={<RoutesPage />} />
          <Route path="monitoring" element={<Monitoring />} />
          <Route path="settings" element={<Settings />} />
          <Route path="docs" element={<Docs />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}

export default App
