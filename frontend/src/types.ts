export interface RestaurantRatingItem {
  id: number
  name: string
  description: string
  created_at: string
}

export interface RestaurantRatingForm {
  id: number
  name: string
  share_token?: string
  published: boolean
  created_at: string
  item_count?: number
  items?: RestaurantRatingItem[]
}

export interface RestaurantRatingScore {
  item_id: number
  item_name: string
  score: number
  evidence: string[]
  passed: boolean
}

export interface RestaurantRatingSubmission {
  id: number
  form_id: number
  view_token?: string
  restaurant: string
  evaluator: string
  form_name?: string
  created_at: string
  scores: RestaurantRatingScore[]
  total_score: number
  max_score: number
  passed_count: number
  total_count: number
}
