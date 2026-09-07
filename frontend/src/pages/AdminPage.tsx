import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api, clearAuth, getUsername } from '../api'
import ItemsTab from '../components/ItemsTab'
import FormsTab from '../components/FormsTab'
import ResultsTab from '../components/ResultsTab'

type Tab = 'items' | 'forms' | 'results'

export default function AdminPage() {
  const navigate = useNavigate()
  const [tab, setTab] = useState<Tab>('items')

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
    <div className="min-h-screen bg-slate-50">
      <header className="bg-white border-b border-slate-200 sticky top-0 z-10">
        <div className="max-w-4xl mx-auto px-3 sm:px-4 py-2 sm:py-0 sm:h-16 flex flex-wrap items-center justify-between gap-x-3 gap-y-2">
          <div className="flex items-center gap-3 sm:gap-8 min-w-0">
            <h1 className="text-base sm:text-lg font-semibold text-slate-900 whitespace-nowrap">
              评估流程表管理
            </h1>
            <nav className="flex gap-0.5 sm:gap-1 min-w-0 overflow-x-auto">
              {(
                [
                  ['items', '评估项'],
                  ['forms', '流程表'],
                  ['results', '评测结果'],
                ] as [Tab, string][]
              ).map(([key, label]) => (
                <button
                  key={key}
                  onClick={() => setTab(key)}
                  className={`px-2 sm:px-3.5 py-1.5 rounded-lg text-sm font-medium whitespace-nowrap transition-colors ${
                    tab === key
                      ? 'bg-indigo-50 text-indigo-700'
                      : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900'
                  }`}
                >
                  {label}
                </button>
              ))}
            </nav>
          </div>
          <div className="flex items-center gap-3 shrink-0">
            <span className="text-sm text-slate-500 whitespace-nowrap hidden sm:inline">
              {getUsername() ? `你好，${getUsername()}` : ''}
            </span>
            <button
              onClick={() => setPwdOpen(true)}
              className="text-sm text-slate-500 hover:text-indigo-700 transition-colors whitespace-nowrap"
            >
              修改密码
            </button>
            <button
              onClick={() => {
                clearAuth()
                navigate('/login', { replace: true })
              }}
              className="text-sm text-slate-500 hover:text-slate-900 transition-colors whitespace-nowrap"
            >
              退出登录
            </button>
          </div>
        </div>
      </header>
      <main className="max-w-4xl mx-auto px-3 sm:px-4 py-6 sm:py-8">
        {tab === 'items' ? <ItemsTab /> : tab === 'forms' ? <FormsTab /> : <ResultsTab />}
      </main>

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
