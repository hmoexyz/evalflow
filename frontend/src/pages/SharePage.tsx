import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api } from '../api'
import type { WorkflowForm } from '../types'

const IMAGE_EXTS = ['.jpg', '.jpeg', '.png', '.gif', '.webp', '.bmp']

function isImage(url: string) {
  const lower = url.toLowerCase()
  return IMAGE_EXTS.some((ext) => lower.endsWith(ext))
}

export default function SharePage() {
  const { token } = useParams<{ token: string }>()
  const navigate = useNavigate()
  const [form, setForm] = useState<WorkflowForm | null>(null)
  const [notFound, setNotFound] = useState(false)
  const [restaurant, setRestaurant] = useState('')
  const [evaluator, setEvaluator] = useState('')
  const [scores, setScores] = useState<Record<number, number>>({})
  const [evidence, setEvidence] = useState<Record<number, string[]>>({})
  const [uploading, setUploading] = useState<Record<number, boolean>>({})
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!token) return
    api
      .getSharedForm(token)
      .then(setForm)
      .catch(() => setNotFound(true))
  }, [token])

  if (notFound) {
    return (
      <div className="min-h-screen bg-slate-50 flex items-center justify-center px-4">
        <div className="text-center">
          <p className="text-5xl mb-4">🔍</p>
          <p className="text-lg font-medium text-slate-900">流程表不存在或未发布</p>
          <p className="mt-2 text-sm text-slate-500">请联系流程表创建者获取有效的分享链接</p>
        </div>
      </div>
    )
  }

  if (!form) {
    return (
      <div className="min-h-screen bg-slate-50 flex items-center justify-center">
        <p className="text-sm text-slate-500">加载中…</p>
      </div>
    )
  }

  const items = form.items ?? []
  const scoredCount = items.filter((it) => scores[it.id] !== undefined).length
  const allScored = scoredCount === items.length
  const infoFilled = restaurant.trim() !== '' && evaluator.trim() !== ''

  async function handleUpload(itemId: number, file: File) {
    setUploading((prev) => ({ ...prev, [itemId]: true }))
    setError('')
    try {
      const { url } = await api.upload(file)
      setEvidence((prev) => ({ ...prev, [itemId]: [...(prev[itemId] ?? []), url] }))
    } catch (err) {
      setError(err instanceof Error ? err.message : '上传失败')
    } finally {
      setUploading((prev) => ({ ...prev, [itemId]: false }))
    }
  }

  function removeEvidence(itemId: number, url: string) {
    setEvidence((prev) => ({
      ...prev,
      [itemId]: (prev[itemId] ?? []).filter((u) => u !== url),
    }))
  }

  async function handleSubmit() {
    if (!allScored || !infoFilled) return
    setSubmitting(true)
    setError('')
    try {
      const { submission: sub } = await api.submit(
        token!,
        restaurant.trim(),
        evaluator.trim(),
        items.map((it) => ({ item_id: it.id, score: scores[it.id], evidence: evidence[it.id] ?? [] })),
      )
      navigate(`/result/${sub.view_token}`, { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : '提交失败')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="min-h-screen bg-slate-50 pb-16">
      <header className="bg-white border-b border-slate-200">
        <div className="max-w-2xl mx-auto px-3 sm:px-4 py-4 sm:py-6">
          <h1 className="text-lg sm:text-xl font-bold text-slate-900 break-words">{form.name}</h1>
          <div className="mt-3 flex items-center gap-3">
            <div className="flex-1 h-2 rounded-full bg-slate-100 overflow-hidden">
              <div
                className="h-full rounded-full bg-indigo-500 transition-all duration-300"
                style={{ width: `${(scoredCount / Math.max(items.length, 1)) * 100}%` }}
              />
            </div>
            <span className="text-xs sm:text-sm text-slate-500 shrink-0 whitespace-nowrap">
              {scoredCount}/{items.length} 项已评分
            </span>
          </div>
        </div>
      </header>

      <main className="max-w-2xl mx-auto px-3 sm:px-4 mt-4 sm:mt-6 space-y-3 sm:space-y-4">
        {error && (
          <div className="rounded-xl bg-red-50 border border-red-200 px-4 py-3 text-sm text-red-600">
            {error}
          </div>
        )}

        <section className="bg-white rounded-2xl shadow-sm border border-slate-200 p-4 sm:p-6">
          <h2 className="text-sm font-semibold text-slate-900 mb-4">测评信息</h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-slate-600 mb-1.5">所在餐厅</label>
              <input
                value={restaurant}
                onChange={(e) => setRestaurant(e.target.value)}
                placeholder="如：海底捞（万达店）"
                className="w-full rounded-lg border border-slate-300 px-3.5 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              />
            </div>
            <div>
              <label className="block text-sm text-slate-600 mb-1.5">测评人</label>
              <input
                value={evaluator}
                onChange={(e) => setEvaluator(e.target.value)}
                placeholder="如：张三"
                className="w-full rounded-lg border border-slate-300 px-3.5 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              />
            </div>
          </div>
        </section>

        {items.map((item, idx) => {
          const score = scores[item.id]
          const ev = evidence[item.id]
          return (
            <section key={item.id} className="bg-white rounded-2xl shadow-sm border border-slate-200 p-4 sm:p-6">
              <div className="flex items-start gap-3">
                <span className="inline-flex items-center justify-center w-7 h-7 rounded-lg bg-indigo-50 text-indigo-700 text-sm font-semibold shrink-0">
                  {idx + 1}
                </span>
                <div className="min-w-0">
                  <h2 className="font-medium text-slate-900">{item.name}</h2>
                  {item.description && (
                    <p className="mt-0.5 text-sm text-slate-500">{item.description}</p>
                  )}
                </div>
              </div>

              <div className="mt-4 sm:mt-5">
                <p className="text-sm text-slate-600 mb-2">评分（0~10 分，6 分及以下为不合格）</p>
                <div className="flex flex-wrap gap-1 sm:gap-1.5">
                  {Array.from({ length: 11 }, (_, v) => v).map((v) => {
                    const selected = score === v
                    const pass = v >= 7
                    return (
                      <button
                        key={v}
                        onClick={() => setScores((prev) => ({ ...prev, [item.id]: v }))}
                        className={`w-8 h-8 sm:w-9 sm:h-9 rounded-lg text-xs sm:text-sm font-medium transition-all ${
                          selected
                            ? pass
                              ? 'bg-emerald-500 text-white shadow-sm'
                              : 'bg-red-500 text-white shadow-sm'
                            : 'bg-slate-50 border border-slate-200 text-slate-600 hover:border-slate-300 hover:bg-slate-100'
                        }`}
                      >
                        {v}
                      </button>
                    )
                  })}
                </div>
                {score !== undefined && (
                  <p className="mt-2 text-sm">
                    {score >= 7 ? (
                      <span className="text-emerald-600 font-medium">合格（{score} 分）</span>
                    ) : (
                      <span className="text-red-600 font-medium">不合格（{score} 分）</span>
                    )}
                  </p>
                )}
              </div>

              <div className="mt-5 border-t border-slate-100 pt-4">
                <p className="text-sm text-slate-600 mb-2">
                  证据（拍照或上传文件，可选）
                  {(ev?.length ?? 0) > 0 && (
                    <span className="ml-1 text-xs text-slate-400">{ev.length} 个</span>
                  )}
                </p>
                {(ev?.length ?? 0) > 0 && (
                  <div className="flex flex-wrap gap-3 mb-3">
                    {ev.map((url) => (
                      <div key={url} className="relative group">
                        {isImage(url) ? (
                          <img
                            src={url}
                            alt="证据"
                            className="h-20 w-20 object-cover rounded-lg border border-slate-200"
                          />
                        ) : (
                          <span className="inline-flex items-center h-20 px-3 rounded-lg bg-slate-50 border border-slate-200 text-sm text-slate-600">
                            文件
                          </span>
                        )}
                        <button
                          onClick={() => removeEvidence(item.id, url)}
                          title="移除"
                          className="absolute -top-1.5 -right-1.5 w-5 h-5 rounded-full bg-red-500 text-white text-xs leading-none hover:bg-red-600 shadow"
                        >
                          ×
                        </button>
                      </div>
                    ))}
                  </div>
                )}
                <label
                  className={`inline-flex items-center gap-2 rounded-lg border border-dashed px-4 py-2.5 text-sm cursor-pointer transition-colors ${
                    uploading[item.id]
                      ? 'border-indigo-300 text-indigo-400 bg-indigo-50/50'
                      : 'border-slate-300 text-slate-600 hover:border-indigo-400 hover:text-indigo-600'
                  }`}
                >
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z"
                    />
                    <path strokeLinecap="round" strokeLinejoin="round" d="M15 13a3 3 0 11-6 0 3 3 0 016 0z" />
                  </svg>
                  {uploading[item.id]
                    ? '上传中…'
                    : (ev?.length ?? 0) > 0
                      ? '继续添加'
                      : '拍照 / 上传文件'}
                  <input
                    type="file"
                    multiple
                    className="hidden"
                    accept="image/*,.pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.txt,.mp4,.mov"
                    onChange={(e) => {
                      const files = e.target.files ? Array.from(e.target.files) : []
                      files.forEach((file) => handleUpload(item.id, file))
                      e.target.value = ''
                    }}
                  />
                </label>
              </div>
            </section>
          )
        })}

        <button
          onClick={handleSubmit}
          disabled={!allScored || !infoFilled || submitting}
          className="w-full rounded-xl bg-indigo-600 py-3.5 text-base font-medium text-white hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors shadow-sm"
        >
          {submitting
            ? '提交中…'
            : !infoFilled
              ? '请填写所在餐厅与测评人'
              : !allScored
                ? `请先完成全部 ${items.length} 项评分`
                : '提交评分并生成统计'}
        </button>
      </main>
    </div>
  )
}
