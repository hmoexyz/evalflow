import { useState } from 'react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { api, clearAuth } from '../api'

export default function AdminPage() {
  const navigate = useNavigate()
  const [sidebarOpen, setSidebarOpen] = useState(false)

  const [pwdOpen, setPwdOpen] = useState(false)
  const [pwdOld, setPwdOld] = useState('')
  const [pwdNew, setPwdNew] = useState('')
  const [pwdConfirm, setPwdConfirm] = useState('')
  const [pwdError, setPwdError] = useState('')
  const [pwdMsg, setPwdMsg] = useState('')
  const [pwdLoading, setPwdLoading] = useState(false)

  function closePwd() {
    setPwdOpen(false)
    setPwdOld('')
    setPwdNew('')
    setPwdConfirm('')
    setPwdError('')
    setPwdMsg('')
    setPwdLoading(false)
  }

  async function handlePwdSubmit(e: React.FormEvent) {
    e.preventDefault()
    setPwdError('')
    setPwdMsg('')
    if (pwdNew.length < 6) {
      setPwdError('新密码至少需要 6 位')
      return
    }
    if (pwdNew !== pwdConfirm) {
      setPwdError('两次输入的新密码不一致')
      return
    }
    setPwdLoading(true)
    try {
      await api.changePassword(pwdOld, pwdNew)
      setPwdOld('')
      setPwdNew('')
      setPwdConfirm('')
      setPwdMsg('密码已修改，下次登录请使用新密码')
    } catch (err) {
      setPwdError(err instanceof Error ? err.message : '修改失败')
    } finally {
      setPwdLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-slate-50 flex">
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-30 bg-slate-900/40 sm:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}
      <aside
        className={`fixed sm:static inset-y-0 left-0 z-40 w-56 shrink-0 bg-white border-r border-slate-200 flex flex-col transform transition-transform duration-200 sm:translate-x-0 ${
          sidebarOpen ? 'translate-x-0' : '-translate-x-full'
        }`}
      >
        <div className="h-14 sm:h-16 flex items-center justify-between gap-2.5 px-5 border-b border-slate-200">
          <div className="flex items-center gap-2.5 min-w-0">
            <div className="w-9 h-9 rounded-xl bg-indigo-600 flex items-center justify-center shrink-0">
              <svg className="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
              </svg>
            </div>
            <span className="font-semibold text-slate-900 whitespace-nowrap">流程管理系统</span>
          </div>
          <button
            onClick={() => setSidebarOpen(false)}
            title="收起菜单"
            className="sm:hidden text-slate-400 hover:text-slate-600 shrink-0"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <nav className="flex-1 p-3 space-y-1">
          <NavLink
            to="/admin/restaurant-rating"
            onClick={() => setSidebarOpen(false)}
            className={({ isActive }) =>
              `w-full flex items-center gap-2.5 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors ${
                isActive ? 'bg-indigo-50 text-indigo-700' : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900'
              }`
            }
          >
            <span className="whitespace-nowrap">餐厅评分</span>
          </NavLink>
        </nav>
        <div className="border-t border-slate-200 p-3 space-y-1">
          <button
            onClick={() => {
              setSidebarOpen(false)
              setPwdOpen(true)
            }}
            className="w-full flex items-center gap-2.5 rounded-lg px-3 py-2.5 text-sm font-medium text-slate-600 hover:bg-slate-100 hover:text-slate-900 transition-colors"
          >
            <svg className="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
            </svg>
            <span className="whitespace-nowrap">修改密码</span>
          </button>
          <button
            onClick={() => {
              clearAuth()
              navigate('/login', { replace: true })
            }}
            className="w-full flex items-center gap-2.5 rounded-lg px-3 py-2.5 text-sm font-medium text-slate-600 hover:bg-slate-100 hover:text-slate-900 transition-colors"
          >
            <svg className="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
            </svg>
            <span className="whitespace-nowrap">退出登录</span>
          </button>
        </div>
      </aside>

      <div className="flex-1 min-w-0 flex flex-col">
        <header className="bg-white border-b border-slate-200 sticky top-0 z-10">
          <div className="px-3 sm:px-6 h-14 sm:h-16 flex items-center gap-0.5 sm:gap-1 overflow-x-auto">
            <button
              onClick={() => setSidebarOpen(true)}
              title="展开菜单"
              className="sm:hidden shrink-0 p-1.5 mr-1 text-slate-500 hover:text-slate-900"
            >
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M4 6h16M4 12h16M4 18h16" />
              </svg>
            </button>
            {(
              [
                ['forms', '流程表'],
                ['items', '评估项'],
                ['results', '评估结果'],
              ] as const
            ).map(([key, label]) => (
              <NavLink
                key={key}
                to={`/admin/restaurant-rating/${key}`}
                className={({ isActive }) =>
                  `px-2 sm:px-3.5 py-1.5 rounded-lg text-sm font-medium whitespace-nowrap transition-colors ${
                    isActive
                      ? 'bg-indigo-50 text-indigo-700'
                      : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900'
                  }`
                }
              >
                {label}
              </NavLink>
            ))}
          </div>
        </header>
        <main className="flex-1 w-full max-w-4xl mx-auto px-3 sm:px-4 py-6 sm:py-8">
          <Outlet />
        </main>
      </div>

      {pwdOpen && (
        <div
          className="fixed inset-0 z-30 bg-slate-900/40 flex items-center justify-center p-4"
          onClick={() => !pwdLoading && closePwd()}
        >
          <div
            className="bg-white rounded-2xl shadow-xl w-full max-w-sm p-6"
            onClick={(e) => e.stopPropagation()}
          >
            <h3 className="text-base font-semibold text-slate-900 mb-5">修改密码</h3>
            <form onSubmit={handlePwdSubmit} className="space-y-3">
              <input
                type="password"
                value={pwdOld}
                onChange={(e) => setPwdOld(e.target.value)}
                placeholder="当前密码"
                autoComplete="current-password"
                autoFocus
                className="w-full rounded-lg border border-slate-300 px-3.5 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              />
              <input
                type="password"
                value={pwdNew}
                onChange={(e) => setPwdNew(e.target.value)}
                placeholder="新密码（至少 6 位）"
                autoComplete="new-password"
                className="w-full rounded-lg border border-slate-300 px-3.5 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              />
              <input
                type="password"
                value={pwdConfirm}
                onChange={(e) => setPwdConfirm(e.target.value)}
                placeholder="再次输入新密码"
                autoComplete="new-password"
                className="w-full rounded-lg border border-slate-300 px-3.5 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              />
              {pwdError && <p className="text-sm text-red-600">{pwdError}</p>}
              {pwdMsg && <p className="text-sm text-emerald-600">{pwdMsg}</p>}
              <div className="flex gap-2 pt-1">
                <button
                  type="submit"
                  disabled={pwdLoading || !pwdOld || !pwdNew || !pwdConfirm}
                  className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                >
                  {pwdLoading ? '提交中…' : '确认修改'}
                </button>
                <button
                  type="button"
                  onClick={closePwd}
                  disabled={pwdLoading}
                  className="rounded-lg border border-slate-300 px-4 py-2 text-sm text-slate-600 hover:bg-slate-50 disabled:opacity-50"
                >
                  关闭
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
