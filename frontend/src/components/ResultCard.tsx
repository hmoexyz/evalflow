import { useRef, useState } from 'react'
import { toPng } from 'html-to-image'
import { QRCodeSVG } from 'qrcode.react'
import type { Submission, WorkflowForm } from '../types'

export default function ResultCard({
  form,
  submission,
}: {
  form: WorkflowForm
  submission: Submission
}) {
  const cardRef = useRef<HTMLDivElement>(null)
  const [exporting, setExporting] = useState(false)
  const [copied, setCopied] = useState(false)

  const passed = submission.passed_count === submission.total_count
  const scorePct = submission.total_count > 0 ? Math.round((submission.total_score / submission.max_score) * 100) : 0
  const resultUrl = submission.view_token ? `${window.location.origin}/result/${submission.view_token}` : null

  async function exportImage() {
    if (!cardRef.current) return
    setExporting(true)
    try {
      const dataUrl = await toPng(cardRef.current, {
        pixelRatio: 2,
        backgroundColor: '#ffffff',
      })
      const a = document.createElement('a')
      a.href = dataUrl
      a.download = `${form.name}_评估结果.png`
      a.click()
    } finally {
      setExporting(false)
    }
  }

  async function copyLink() {
    if (!resultUrl) return
    await navigator.clipboard.writeText(resultUrl)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="bg-white rounded-2xl shadow-sm border border-slate-200 p-4 sm:p-6">
      <div ref={cardRef} className="p-1.5 sm:p-2">
        <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-2 mb-5 sm:mb-6">
          <div className="min-w-0">
            <h2 className="text-lg sm:text-xl font-bold text-slate-900">{form.name}</h2>
            <p className="mt-1 text-xs sm:text-sm text-slate-500">
              {submission.restaurant && (
                <span>餐厅：{submission.restaurant} · </span>
              )}
              {submission.evaluator && <span>测评人：{submission.evaluator} · </span>}
              提交于 {submission.created_at} · 编号 #{submission.id}
            </p>
          </div>
          <div
            className={`flex items-center gap-1.5 rounded-full px-3 sm:px-4 py-1.5 sm:py-2 text-xs sm:text-sm font-medium whitespace-nowrap ${
              passed ? 'bg-emerald-50 text-emerald-700' : 'bg-red-50 text-red-600'
            }`}
          >
            <span className="text-sm sm:text-lg">{passed ? '✓' : '✗'}</span>
            {passed ? '总体合格' : '总体不合格'}
          </div>
        </div>

        <div className="grid grid-cols-3 gap-2 sm:gap-4 mb-5 sm:mb-6">
          <div className="rounded-xl bg-slate-50 p-2.5 sm:p-4 text-center">
            <p className="text-2xl sm:text-3xl font-bold text-slate-900">
              {submission.total_score}
              <span className="text-sm sm:text-base font-normal text-slate-400">/{submission.max_score}</span>
            </p>
            <p className="mt-1 text-xs sm:text-sm text-slate-500">总得分 / 满分</p>
          </div>
          <div className="rounded-xl bg-slate-50 p-2.5 sm:p-4 text-center">
            <p className="text-2xl sm:text-3xl font-bold text-slate-900">
              {submission.passed_count}
              <span className="text-sm sm:text-base font-normal text-slate-400">/{submission.total_count}</span>
            </p>
            <p className="mt-1 text-xs sm:text-sm text-slate-500">合格项数 / 总项数</p>
          </div>
          <div className="rounded-xl bg-slate-50 p-2.5 sm:p-4 text-center">
            <p className="text-2xl sm:text-3xl font-bold text-slate-900">{scorePct}%</p>
            <p className="mt-1 text-xs sm:text-sm text-slate-500">得分率</p>
          </div>
        </div>

        <div className="mb-6 h-2 rounded-full bg-slate-100 overflow-hidden">
          <div
            className={`h-full rounded-full ${passed ? 'bg-emerald-500' : 'bg-red-500'}`}
            style={{ width: `${scorePct}%` }}
          />
        </div>

        <ul className="divide-y divide-slate-100">
          {submission.scores.map((sc, idx) => (
            <li key={sc.item_id} className="py-3 flex items-center justify-between gap-3">
              <div className="min-w-0 flex items-center gap-3">
                <span
                  className={`inline-flex items-center justify-center w-6 h-6 rounded-full text-xs font-medium shrink-0 ${
                    sc.passed ? 'bg-emerald-100 text-emerald-700' : 'bg-red-100 text-red-600'
                  }`}
                >
                  {sc.passed ? '✓' : '✗'}
                </span>
                <div className="min-w-0">
                  <p className="text-sm font-medium text-slate-900 truncate">
                    {idx + 1}. {sc.item_name}
                  </p>
                  {(sc.evidence ?? []).length > 0 && (
                    <div className="mt-0.5 flex flex-wrap gap-x-3 gap-y-0.5">
                      {(sc.evidence ?? []).map((url, i) => (
                        <a
                          key={url}
                          href={url}
                          target="_blank"
                          rel="noreferrer"
                          className="text-xs text-indigo-600 hover:text-indigo-800"
                        >
                          附件 {i + 1}
                        </a>
                      ))}
                    </div>
                  )}
                </div>
              </div>
              <span
                className={`text-lg font-semibold shrink-0 ${
                  sc.passed ? 'text-emerald-600' : 'text-red-600'
                }`}
              >
                {sc.score}
              </span>
            </li>
          ))}
        </ul>
        <p className="mt-4 text-xs text-slate-400">合格标准：每项得分 ≥ 7 分（6 分及以下为不合格）</p>

        {resultUrl && (
          <div className="mt-5 sm:mt-6 pt-5 sm:pt-6 border-t border-slate-100">
            <div className="flex items-center gap-3 sm:gap-4">
              <div className="bg-white border border-slate-200 rounded-lg p-2 shrink-0">
                <QRCodeSVG value={resultUrl} size={84} />
              </div>
              <div className="min-w-0">
                <p className="text-sm font-medium text-slate-900">结果查看链接</p>
                <p className="mt-1 text-xs text-slate-500 break-all leading-relaxed">{resultUrl}</p>
                <p className="mt-1 text-xs text-slate-400">扫描二维码或打开链接，可查看本次评估结果与证据附件</p>
              </div>
            </div>
          </div>
        )}
      </div>

      <div className="mt-4 flex flex-wrap justify-center gap-2">
        {resultUrl && (
          <button
            onClick={copyLink}
            className="inline-flex items-center gap-2 rounded-lg border border-slate-300 px-4 sm:px-5 py-2.5 text-sm font-medium text-slate-700 hover:bg-slate-50 transition-colors"
          >
            {copied ? '已复制' : '复制链接'}
          </button>
        )}
        <button
          onClick={exportImage}
          disabled={exporting}
          className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 sm:px-5 py-2.5 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50 transition-colors"
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
            />
          </svg>
          {exporting ? '生成中…' : '导出为图片'}
        </button>
      </div>
    </div>
  )
}
