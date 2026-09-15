import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { api } from '../api'
import type { RestaurantRatingSubmission, RestaurantRatingForm } from '../types'
import ResultCard from '../components/ResultCard'

export default function ResultPage() {
  const { token } = useParams<{ token: string }>()
  const [data, setData] = useState<{ form: RestaurantRatingForm; submission: RestaurantRatingSubmission } | null>(null)
  const [notFound, setNotFound] = useState(false)

  useEffect(() => {
    if (!token) return
    api
      .getRestaurantRatingResult(token)
      .then(setData)
      .catch(() => setNotFound(true))
  }, [token])

  if (notFound) {
    return (
      <div className="min-h-screen bg-slate-50 flex items-center justify-center px-4">
        <div className="text-center">
          <p className="text-5xl mb-4">🔍</p>
          <p className="text-lg font-medium text-slate-900">结果不存在</p>
          <p className="mt-2 text-sm text-slate-500">链接可能已失效，请联系流程表创建者</p>
        </div>
      </div>
    )
  }

  if (!data) {
    return (
      <div className="min-h-screen bg-slate-50 flex items-center justify-center">
        <p className="text-sm text-slate-500">加载中…</p>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-slate-50 py-6 sm:py-10 px-3 sm:px-4">
      <div className="max-w-2xl mx-auto space-y-4 sm:space-y-6">
        <div className="text-center px-2">
          <h1 className="text-xl font-semibold text-slate-900">评估结果</h1>
          <p className="mt-1 text-sm text-slate-500">
            {data.form.name} · 编号 #{data.submission.id}
          </p>
        </div>
        <ResultCard form={data.form} submission={data.submission} />
      </div>
    </div>
  )
}
