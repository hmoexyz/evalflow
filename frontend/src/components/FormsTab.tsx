import { useEffect, useState } from 'react'
import { api } from '../api'
import type { RestaurantRatingItem, RestaurantRatingSubmission, RestaurantRatingForm } from '../types'

export default function FormsTab() {
  const [forms, setForms] = useState<RestaurantRatingForm[]>([])
  const [items, setItems] = useState<RestaurantRatingItem[]>([])
  const [error, setError] = useState('')

  const [editing, setEditing] = useState<RestaurantRatingForm | null>(null)
  const [formName, setFormName] = useState('')
  const [selectedIds, setSelectedIds] = useState<number[]>([])
  const [itemQuery, setItemQuery] = useState('')
  const [newItemOpen, setNewItemOpen] = useState(false)
  const [newItemName, setNewItemName] = useState('')
  const [newItemDesc, setNewItemDesc] = useState('')
  const [creatingItem, setCreatingItem] = useState(false)
  const [viewing, setViewing] = useState<RestaurantRatingForm | null>(null)
  const [submissions, setSubmissions] = useState<RestaurantRatingSubmission[]>([])
  const [copiedId, setCopiedId] = useState<number | null>(null)

  async function load() {
    const [f, i] = await Promise.all([api.listRestaurantRatingForms(), api.listRestaurantRatingItems()])
    setForms(f)
    setItems(i)
  }

  useEffect(() => {
    load().catch(() => setError('加载失败'))
  }, [])

  function openCreate() {
    setEditing({ id: 0, name: '', published: false, created_at: '' })
    setFormName('')
    setSelectedIds([])
    setItemQuery('')
    resetNewItem()
  }

  function openEdit(form: RestaurantRatingForm) {
    setEditing(form)
    setFormName(form.name)
    setSelectedIds((form.items ?? []).map((it) => it.id))
    setItemQuery('')
    resetNewItem()
  }

  function resetNewItem() {
    setNewItemOpen(false)
    setNewItemName('')
    setNewItemDesc('')
    setCreatingItem(false)
  }

  async function handleCreateItem() {
    setError('')
    setCreatingItem(true)
    try {
      const item = await api.createRestaurantRatingItem(newItemName.trim(), newItemDesc.trim())
      setItems((prev) => [item, ...prev])
      setSelectedIds((prev) => [...prev, item.id])
      resetNewItem()
    } catch (err) {
      setError(err instanceof Error ? err.message : '创建评估项失败')
      setCreatingItem(false)
    }
  }

  const visibleItems = itemQuery.trim()
    ? items.filter((it) => `${it.name} ${it.description}`.toLowerCase().includes(itemQuery.trim().toLowerCase()))
    : items

  function toggleItem(id: number) {
    setSelectedIds((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]))
  }

  async function handleSave() {
    setError('')
    try {
      if (editing?.id) {
        await api.updateRestaurantRatingForm(editing.id, formName.trim(), selectedIds)
      } else {
        await api.createRestaurantRatingForm(formName.trim(), selectedIds)
      }
      setEditing(null)
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存失败')
    }
  }

  async function handleDelete(id: number) {
    if (!window.confirm('确定删除该流程表吗？相关的提交记录也会被删除。')) return
    setError('')
    try {
      await api.deleteRestaurantRatingForm(id)
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : '删除失败')
    }
  }

  async function handlePublish(form: RestaurantRatingForm) {
    setError('')
    try {
      if (form.published) {
        await api.unpublishRestaurantRatingForm(form.id)
      } else {
        await api.publishRestaurantRatingForm(form.id)
      }
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : '操作失败')
    }
  }

  async function openSubmissions(form: RestaurantRatingForm) {
    setViewing(form)
    setSubmissions(await api.listRestaurantRatingSubmissions(form.id))
  }

  function shareUrl(form: RestaurantRatingForm) {
    return `${window.location.origin}/share/${form.share_token}`
  }

  async function copyShare(form: RestaurantRatingForm) {
    await navigator.clipboard.writeText(shareUrl(form))
    setCopiedId(form.id)
    setTimeout(() => setCopiedId((id) => (id === form.id ? null : id)), 2000)
  }

  return (
    <div className="space-y-6">
      {error && <p className="text-sm text-red-600">{error}</p>}

      {editing && (
        <section className="bg-white rounded-2xl shadow-sm border border-slate-200 p-4 sm:p-6">
          <h2 className="text-base font-semibold text-slate-900 mb-4">
            {editing.id ? '编辑流程表' : '新建流程表'}
          </h2>
          <input
            value={formName}
            onChange={(e) => setFormName(e.target.value)}
            placeholder="流程表名称，如：月度巡检"
            className="w-full rounded-lg border border-slate-300 px-3.5 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          />
          <div className="mt-4">
            <p className="text-sm text-slate-600 mb-2">包含的评估项（按勾选顺序排列）</p>
            {items.length === 0 ? (
              <p className="text-sm text-slate-400 py-4 text-center border border-dashed border-slate-200 rounded-lg">
                暂无评估项，可在下方直接新建
              </p>
            ) : (
              <>
                <div className="relative mb-2">
                  <svg
                    className="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    strokeWidth={2}
                  >
                    <path strokeLinecap="round" strokeLinejoin="round" d="M21 21l-4.35-4.35M17 10.5a6.5 6.5 0 11-13 0 6.5 6.5 0 0113 0z" />
                  </svg>
                  <input
                    value={itemQuery}
                    onChange={(e) => setItemQuery(e.target.value)}
                    placeholder="搜索评估项名称或说明…"
                    className="w-full rounded-lg border border-slate-300 pl-9 pr-3.5 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                  />
                </div>
                {visibleItems.length === 0 ? (
                  <p className="text-sm text-slate-400 py-4 text-center border border-dashed border-slate-200 rounded-lg">
                    没有匹配的评估项
                  </p>
                ) : (
                  <div className="grid grid-cols-1 gap-2">
                    {visibleItems.map((item) => (
                      <label
                        key={item.id}
                        className={`flex items-center gap-3 rounded-lg border px-3 py-2.5 cursor-pointer transition-colors ${
                          selectedIds.includes(item.id)
                            ? 'border-indigo-500 bg-indigo-50'
                            : 'border-slate-200 hover:border-slate-300'
                        }`}
                      >
                        <input
                          type="checkbox"
                          checked={selectedIds.includes(item.id)}
                          onChange={() => toggleItem(item.id)}
                          className="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
                        />
                        <div className="min-w-0">
                          <p className="text-sm font-medium text-slate-900 truncate">{item.name}</p>
                          {item.description && (
                            <p className="text-xs text-slate-500 truncate">{item.description}</p>
                          )}
                        </div>
                      </label>
                    ))}
                  </div>
                )}
              </>
            )}
            <p className="mt-2 text-xs text-slate-400">已选 {selectedIds.length} 项</p>

            <div className="mt-3 rounded-lg border border-dashed border-slate-300 p-3">
              {newItemOpen ? (
                <div className="space-y-2">
                  <input
                    value={newItemName}
                    onChange={(e) => setNewItemName(e.target.value)}
                    placeholder="评估项名称，如：卫生状况"
                    autoFocus
                    className="w-full rounded-lg border border-slate-300 px-3.5 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                  />
                  <textarea
                    value={newItemDesc}
                    onChange={(e) => setNewItemDesc(e.target.value)}
                    placeholder="说明（可选）"
                    rows={2}
                    className="w-full rounded-lg border border-slate-300 px-3.5 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 resize-none"
                  />
                  <div className="flex gap-2">
                    <button
                      type="button"
                      onClick={handleCreateItem}
                      disabled={!newItemName.trim() || creatingItem}
                      className="rounded-lg bg-indigo-600 px-3 py-1.5 text-sm text-white hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                    >
                      {creatingItem ? '添加中…' : '添加并勾选'}
                    </button>
                    <button
                      type="button"
                      onClick={resetNewItem}
                      disabled={creatingItem}
                      className="rounded-lg border border-slate-300 px-3 py-1.5 text-sm text-slate-600 hover:bg-slate-50 disabled:opacity-50"
                    >
                      取消
                    </button>
                  </div>
                </div>
              ) : (
                <button
                  type="button"
                  onClick={() => setNewItemOpen(true)}
                  className="inline-flex items-center gap-1.5 text-sm font-medium text-indigo-600 hover:text-indigo-800"
                >
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
                  </svg>
                  新建评估项
                </button>
              )}
            </div>
          </div>
          <div className="mt-5 flex gap-2">
            <button
              onClick={handleSave}
              disabled={!formName.trim() || selectedIds.length === 0}
              className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              保存
            </button>
            <button
              onClick={() => setEditing(null)}
              className="rounded-lg border border-slate-300 px-4 py-2 text-sm text-slate-600 hover:bg-slate-50"
            >
              取消
            </button>
          </div>
        </section>
      )}

      <div className="flex items-center justify-between gap-3 flex-wrap">
        <h2 className="text-base font-semibold text-slate-900">
          流程表列表
          <span className="ml-2 text-sm font-normal text-slate-500">{forms.length} 个</span>
        </h2>
        <button
          onClick={openCreate}
          className="inline-flex items-center gap-1.5 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 transition-colors whitespace-nowrap"
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
          </svg>
          新建流程表
        </button>
      </div>

      <section className="bg-white rounded-2xl shadow-sm border border-slate-200 p-4 sm:p-6">
        {forms.length === 0 ? (
          <p className="text-sm text-slate-500 py-8 text-center">暂无流程表</p>
        ) : (
          <ul className="divide-y divide-slate-100">
            {forms.map((form) => (
              <li key={form.id} className="py-4">
                <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between sm:gap-4">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <p className="font-medium text-slate-900">{form.name}</p>
                      {form.published ? (
                        <span className="inline-flex items-center rounded-full bg-emerald-50 px-2 py-0.5 text-xs font-medium text-emerald-700">
                          已发布
                        </span>
                      ) : (
                        <span className="inline-flex items-center rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-500">
                          未发布
                        </span>
                      )}
                    </div>
                    <p className="mt-0.5 text-sm text-slate-500">
                      {form.item_count} 个评估项 · 创建于 {form.created_at}
                    </p>
                    {form.published && (
                      <div className="mt-2 flex items-center gap-2">
                        <a
                          href={shareUrl(form)}
                          target="_blank"
                          rel="noreferrer"
                          title={shareUrl(form)}
                          className="inline-flex items-center gap-1.5 rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-indigo-700 transition-colors"
                        >
                          <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
                            />
                          </svg>
                          打开
                        </a>
                        <button
                          onClick={() => copyShare(form)}
                          title="复制链接"
                          className={`inline-flex items-center gap-1 rounded-md border px-2 py-1.5 text-xs shrink-0 transition-colors ${
                            copiedId === form.id
                              ? 'border-emerald-200 bg-emerald-50 text-emerald-700'
                              : 'border-slate-200 text-slate-600 hover:bg-slate-50 hover:border-slate-300'
                          }`}
                        >
                          {copiedId === form.id ? (
                            '已复制'
                          ) : (
                            <>
                              <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                                <path
                                  strokeLinecap="round"
                                  strokeLinejoin="round"
                                  d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
                                />
                              </svg>
                              复制
                            </>
                          )}
                        </button>
                      </div>
                    )}
                  </div>
                  <div className="flex flex-wrap items-center gap-x-4 gap-y-1 sm:gap-x-3 sm:justify-end shrink-0">
                    <button
                      onClick={() => handlePublish(form)}
                      className="text-sm text-emerald-600 hover:text-emerald-800"
                    >
                      {form.published ? '取消发布' : '发布'}
                    </button>
                    <button
                      onClick={() => openSubmissions(form)}
                      className="text-sm text-indigo-600 hover:text-indigo-800"
                    >
                      数据
                    </button>
                    <button
                      onClick={() => openEdit(form)}
                      className="text-sm text-slate-600 hover:text-slate-900"
                    >
                      编辑
                    </button>
                    <button
                      onClick={() => handleDelete(form.id)}
                      className="text-sm text-red-600 hover:text-red-800"
                    >
                      删除
                    </button>
                  </div>
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>

      {viewing && (
        <div
          className="fixed inset-0 z-20 bg-slate-900/40 flex items-start justify-center overflow-y-auto p-4"
          onClick={() => setViewing(null)}
        >
          <div
            className="bg-white rounded-2xl shadow-xl w-full max-w-2xl my-4 sm:my-8 p-4 sm:p-6"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-start justify-between gap-3 mb-4">
              <h3 className="text-base font-semibold text-slate-900 break-words">
                {viewing.name} · 提交记录
                <span className="ml-2 text-sm font-normal text-slate-500">
                  {submissions.length} 条
                </span>
              </h3>
              <button
                onClick={() => setViewing(null)}
                className="text-slate-400 hover:text-slate-600 text-xl leading-none"
              >
                ×
              </button>
            </div>
            {submissions.length === 0 ? (
              <p className="text-sm text-slate-500 py-8 text-center">暂无提交记录</p>
            ) : (
              <ul className="space-y-4">
                {submissions.map((sub) => (
                  <li key={sub.id} className="rounded-xl border border-slate-200 p-4">
                    <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1 mb-3">
                      <p className="text-sm text-slate-500">
                        #{sub.id} · {sub.created_at}
                        {sub.view_token && (
                          <a
                            href={`/result/${sub.view_token}`}
                            target="_blank"
                            rel="noreferrer"
                            className="ml-2 text-indigo-600 hover:text-indigo-800"
                          >
                            结果链接
                          </a>
                        )}
                      </p>
                      <p className="text-sm font-medium text-slate-900">
                        {sub.total_score}/{sub.max_score} 分 · 合格{' '}
                        <span className={sub.passed_count === sub.total_count ? 'text-emerald-600' : 'text-amber-600'}>
                          {sub.passed_count}/{sub.total_count}
                        </span>
                      </p>
                    </div>
                    <ul className="divide-y divide-slate-100">
                      {sub.scores.map((sc) => (
                        <li key={sc.item_id} className="py-2 flex items-center justify-between gap-3">
                          <div className="min-w-0 flex items-center gap-2">
                            <span
                              className={`inline-flex items-center justify-center w-5 h-5 rounded-full text-xs font-medium shrink-0 ${
                                sc.passed ? 'bg-emerald-100 text-emerald-700' : 'bg-red-100 text-red-600'
                              }`}
                            >
                              {sc.passed ? '✓' : '✗'}
                            </span>
                            <span className="text-sm text-slate-800 truncate">{sc.item_name}</span>
                            {(sc.evidence ?? []).length > 0 && (
                              <div className="flex gap-2 shrink-0">
                                {(sc.evidence ?? []).map((url, i) => (
                                  <a
                                    key={url}
                                    href={url}
                                    target="_blank"
                                    rel="noreferrer"
                                    className="text-xs text-indigo-600 hover:text-indigo-800"
                                  >
                                    附件{i + 1}
                                  </a>
                                ))}
                              </div>
                            )}
                          </div>
                          <span
                            className={`text-sm font-medium shrink-0 ${
                              sc.passed ? 'text-emerald-600' : 'text-red-600'
                            }`}
                          >
                            {sc.score} 分
                          </span>
                        </li>
                      ))}
                    </ul>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
