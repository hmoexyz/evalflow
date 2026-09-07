import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api'
import type { WorkflowForm } from '../types'

export default function PublicFormsPage() {
  const [forms, setForms] = useState<WorkflowForm[] | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    api
      .listPublishedForms()
      .then(setForms)
      .catch(() => setError('加载失败，请稍后重试'))
  }, [])

  return (
    <div className="min-h-screen bg-slate-50">
      <header className="bg-white border-b border-slate-200">
        <div className="max-w-2xl mx-auto px-3 sm:px-4 py-3 sm:py-4 flex items-center justify-between gap-3">
          <div className="flex items-center gap-2.5">
            <div className="w-9 h-9 rounded-xl bg-indigo-600 flex items-center justify-center">
              <svg className="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
              </svg>
            </div>
            <span className="font-semibold text-slate-900">评估流程表</span>
          </div>
          <Link
            to="/login"
            className="rounded-lg border border-slate-300 px-3.5 py-1.5 text-sm font-medium text-slate-600 hover:bg-slate-50 hover:text-slate-900 transition-colors whitespace-nowrap"
          >
            登录 / 注册
          </Link>
        </div>
      </header>

      <main className="max-w-2xl mx-auto px-3 sm:px-4 py-6 sm:py-8 space-y-4">
        <div>
          <h1 className="text-lg sm:text-xl font-bold text-slate-900">已发布的流程表</h1>
          <p className="mt-1 text-sm text-slate-500">无需登录，选择一个流程表开始填写</p>
        </div>

        {error && <p className="text-sm text-red-600">{error}</p>}

        {forms === null ? (
          <p className="text-sm text-slate-500 py-10 text-center">加载中…</p>
        ) : forms.length === 0 ? (
          <div className="bg-white rounded-2xl shadow-sm border border-slate-200 px-4 py-12 text-center">
            <p className="text-4xl mb-3">📋</p>
            <p className="text-sm text-slate-500">暂无已发布的流程表</p>
          </div>
        ) : (
          <ul className="space-y-3">
            {forms.map((form) => (
              <li
                key={form.id}
                className="bg-white rounded-2xl shadow-sm border border-slate-200 p-4 sm:p-5 flex items-center justify-between gap-4"
              >
                <div className="min-w-0">
                  <p className="font-medium text-slate-900 break-words">{form.name}</p>
                  <p className="mt-1 text-sm text-slate-500">
                    {form.item_count ?? 0} 个评估项 · 发布于 {form.created_at}
                  </p>
                </div>
                {form.share_token && (
                  <Link
                    to={`/share/${form.share_token}`}
                    className="shrink-0 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 transition-colors"
                  >
                    开始填写
                  </Link>
                )}
              </li>
            ))}
          </ul>
        )}
      </main>
    </div>
  )
}
