export interface EvaluationItem {
  id: number
  name: string
  description: string
  created_at: string
}

export interface WorkflowForm {
  id: number
  name: string
  share_token?: string
  published: boolean
  created_at: string
  item_count?: number
  items?: EvaluationItem[]
}

export interface ScoreEntry {
  item_id: number
  item_name: string
  score: number
  evidence: string[]
  passed: boolean
}

export interface Submission {
  id: number
  form_id: number
  view_token?: string
  restaurant: string
  evaluator: string
  form_name?: string
  created_at: string
  scores: ScoreEntry[]
  total_score: number
  max_score: number
  passed_count: number
  total_count: number
}
