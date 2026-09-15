package main

// restaurantRatingPassScore is the minimum score for a restaurant item to be
// considered 合格 (7-10). 6分及以下为不合格.
const restaurantRatingPassScore = 7

// User is a registered account. Restaurant items and forms are scoped to one user.
type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	CreatedAt string `json:"created_at"`
}

// RestaurantRatingItem is an evaluation item belonging to the 餐厅评分 subsystem.
type RestaurantRatingItem struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

// RestaurantRatingForm is a workflow form belonging to the 餐厅评分 subsystem.
type RestaurantRatingForm struct {
	ID         int64                  `json:"id"`
	Name       string                 `json:"name"`
	ShareToken string                 `json:"share_token,omitempty"`
	Published  bool                   `json:"published"`
	CreatedAt  string                 `json:"created_at"`
	ItemCount  int                    `json:"item_count,omitempty"`
	Items      []RestaurantRatingItem `json:"items,omitempty"`
}

type RestaurantRatingScore struct {
	ItemID   int64    `json:"item_id"`
	ItemName string   `json:"item_name"`
	Score    int      `json:"score"`
	Evidence []string `json:"evidence"`
	Passed   bool     `json:"passed"`
}

type RestaurantRatingSubmission struct {
	ID          int64                   `json:"id"`
	FormID      int64                   `json:"form_id"`
	ViewToken   string                  `json:"view_token,omitempty"`
	Restaurant  string                  `json:"restaurant"`
	Evaluator   string                  `json:"evaluator"`
	FormName    string                  `json:"form_name,omitempty"`
	CreatedAt   string                  `json:"created_at"`
	Scores      []RestaurantRatingScore `json:"scores"`
	TotalScore  int                     `json:"total_score"`
	MaxScore    int                     `json:"max_score"`
	PassedCount int                     `json:"passed_count"`
	TotalCount  int                     `json:"total_count"`
}

func computeRestaurantRatingStats(scores []RestaurantRatingScore) (total, max, passed, count int) {
	max = len(scores) * 10
	count = len(scores)
	for _, sc := range scores {
		total += sc.Score
		if sc.Passed {
			passed++
		}
	}
	return
}
