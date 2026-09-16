import { BrowserRouter, Navigate, Route, Routes, useNavigate } from 'react-router-dom'
import { getToken } from './api'
import AdminPage from './pages/AdminPage'
import LoginPage from './pages/LoginPage'
import PublicFormsPage from './pages/PublicFormsPage'
import ResultPage from './pages/ResultPage'
import SharePage from './pages/SharePage'
import ItemsTab from './components/ItemsTab'
import FormsTab from './components/FormsTab'
import ResultsTab from './components/ResultsTab'

function Home() {
  // 已登录进后台；访客在首页浏览所有已发布的流程表
  return getToken() ? <Navigate to="/admin" replace /> : <PublicFormsPage />
}

function LoginRoute() {
  const navigate = useNavigate()
  if (getToken()) return <Navigate to="/admin" replace />
  return <LoginPage onLogin={() => navigate('/admin', { replace: true })} />
}

function AdminRoute() {
  if (!getToken()) return <Navigate to="/login" replace />
  return <AdminPage />
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/login" element={<LoginRoute />} />
        <Route path="/admin" element={<AdminRoute />}>
          <Route index element={<Navigate to="restaurant-rating/forms" replace />} />
          <Route path="restaurant-rating">
            <Route index element={<Navigate to="forms" replace />} />
            <Route path="forms" element={<FormsTab />} />
            <Route path="items" element={<ItemsTab />} />
            <Route path="results" element={<ResultsTab />} />
          </Route>
        </Route>
        <Route path="/share/:token" element={<SharePage />} />
        <Route path="/result/:token" element={<ResultPage />} />
        <Route path="*" element={<Home />} />
      </Routes>
    </BrowserRouter>
  )
}
