import { useEffect, useState } from 'react'
import { api } from '../api'
import type { RestaurantRatingSubmission, RestaurantRatingForm } from '../types'

export default function ResultsTab() {
  const [subs, setSubs] = useState<RestaurantRatingSubmission[]>([])
  const [forms, setForms] = useState<RestaurantRatingForm[]>([])
  const [filter, setFilter] = useState<number>(0)
  const [error, setError] = useState('')

  async function load() {
    const [s, f] = await Promise.all([api.listAllRestaurantRatingSubmissions(), api.listRestaurantRatingForms()])
    setSubs(s)
    setForms(f)
  }

  useEffect(() => {
    load().catch(() => setError('加载失败'))
  }, [])

  const filtered = filter ? subs.filter((sub) => sub.form_id === filter) : subs

  return (
    <div className="space-y-6">
      {error && <p className="text-sm text-red-600">{error}</p>}

      <section className="bg-white rounded-2xl shadow-sm border border-slate-200 p-4 sm:p-6">
        <div className="flex items-center justify-between mb-4 gap-4 flex-wrap">
          <h2 className="text-base font-semibold text-slate-900">
            评估结果列表
            <span className="ml-2 text-sm font-normal text-slate-500">{filtered.length} 条</span>
          </h2>
          {forms.length > 0 && (
            <select
              value={filter}
              onChange={(e) => setFilter(Number(e.target.value))}
              className="rounded-lg border border-slate-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 bg-white"
            >
              <option value={0}>全部流程表</option>
              {forms.map((form) => (
                <option key={form.id} value={form.id}>
                  {form.name}
                </option>
              ))}
            </select>
          )}
        </div>

        {filtered.length === 0 ? (
          <p className="text-sm text-slate-500 py-8 text-center">暂无评估结果</p>
        ) : (
          <ul className="divide-y divide-slate-100">
            {filtered.map((sub) => {
              const passed = sub.passed_count === sub.total_count
              return (
                <li key={sub.id} className="py-4">
                  <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between sm:gap-4">
                    <div className="min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <p className="font-medium text-slate-900 break-words">{sub.restaurant}</p>
                        <span
                          className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${
                            passed ? 'bg-emerald-50 text-emerald-700' : 'bg-red-50 text-red-600'
                          }`}
                        >
                          {passed ? '合格' : '不合格'}
                        </span>
                      </div>
                      <p className="mt-1 text-sm text-slate-500">
                        测评人：{sub.evaluator} · {sub.form_name} · {sub.created_at} · #{sub.id}
                      </p>
                      <p className="mt-1 text-sm text-slate-600">
                        得分{' '}
                        <span className="font-medium text-slate-900">
                          {sub.total_score}/{sub.max_score}
                        </span>
                        {' · '}合格{' '}
                        <span className={passed ? 'font-medium text-emerald-600' : 'font-medium text-red-600'}>
                          {sub.passed_count}/{sub.total_count}
                        </span>
                      </p>
                    </div>
                    <div className="flex gap-3 shrink-0">
                      {sub.view_token && (
                        <a
                          href={`/result/${sub.view_token}`}
                          target="_blank"
                          rel="noreferrer"
                          className="text-sm text-indigo-600 hover:text-indigo-800"
                        >
                          查看结果
                        </a>
                      )}
                    </div>
                  </div>
                </li>
              )
            })}
          </ul>
        )}
      </section>
    </div>
  )
}
