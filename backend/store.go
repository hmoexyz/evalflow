package main

import (
	"database/sql"
	"strings"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func openStore(path string) (*Store, error) {
	dsn := "file:" + path + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	// rename legacy tables to the 餐厅评分-namespaced names, so other
	// subsystems added later cannot collide with them
	for _, r := range [][2]string{
		{"evaluation_items", "restaurant_rating_evaluation_items"},
		{"workflow_forms", "restaurant_rating_workflow_forms"},
		{"form_items", "restaurant_rating_form_items"},
		{"submissions", "restaurant_rating_submissions"},
		{"submission_scores", "restaurant_rating_submission_scores"},
		// databases that used the earlier "restaurant_" prefix
		{"restaurant_evaluation_items", "restaurant_rating_evaluation_items"},
		{"restaurant_workflow_forms", "restaurant_rating_workflow_forms"},
		{"restaurant_form_items", "restaurant_rating_form_items"},
		{"restaurant_submissions", "restaurant_rating_submissions"},
		{"restaurant_submission_scores", "restaurant_rating_submission_scores"},
	} {
		if err := s.renameTableIfExists(r[0], r[1]); err != nil {
			return err
		}
	}

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL COLLATE NOCASE UNIQUE,
			password_hash TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
		)`,
		`CREATE TABLE IF NOT EXISTS restaurant_rating_evaluation_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL DEFAULT 0,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
		)`,
		`CREATE TABLE IF NOT EXISTS restaurant_rating_workflow_forms (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL DEFAULT 0,
			name TEXT NOT NULL,
			share_token TEXT UNIQUE,
			created_at TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
		)`,
		`CREATE TABLE IF NOT EXISTS restaurant_rating_form_items (
			form_id INTEGER NOT NULL REFERENCES restaurant_rating_workflow_forms(id) ON DELETE CASCADE,
			item_id INTEGER NOT NULL REFERENCES restaurant_rating_evaluation_items(id) ON DELETE CASCADE,
			position INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (form_id, item_id)
		)`,
		`CREATE TABLE IF NOT EXISTS restaurant_rating_submissions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			form_id INTEGER NOT NULL REFERENCES restaurant_rating_workflow_forms(id) ON DELETE CASCADE,
			view_token TEXT,
			restaurant TEXT NOT NULL DEFAULT '',
			evaluator TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
		)`,
		`CREATE TABLE IF NOT EXISTS restaurant_rating_submission_scores (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			submission_id INTEGER NOT NULL REFERENCES restaurant_rating_submissions(id) ON DELETE CASCADE,
			item_id INTEGER NOT NULL REFERENCES restaurant_rating_evaluation_items(id),
			score INTEGER NOT NULL CHECK (score >= 0 AND score <= 10),
			evidence TEXT NOT NULL DEFAULT '',
			UNIQUE (submission_id, item_id)
		)`,
		`CREATE TABLE IF NOT EXISTS uploads (
			id TEXT PRIMARY KEY,
			filename TEXT NOT NULL DEFAULT '',
			content_type TEXT NOT NULL DEFAULT '',
			size INTEGER NOT NULL DEFAULT 0,
			data BLOB NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	// databases created before later features: add missing columns
	for _, c := range []struct{ table, col, decl string }{
		{"restaurant_rating_evaluation_items", "user_id", "INTEGER NOT NULL DEFAULT 0"},
		{"restaurant_rating_workflow_forms", "user_id", "INTEGER NOT NULL DEFAULT 0"},
		{"restaurant_rating_submissions", "view_token", "TEXT"},
		{"restaurant_rating_submissions", "restaurant", "TEXT NOT NULL DEFAULT ''"},
		{"restaurant_rating_submissions", "evaluator", "TEXT NOT NULL DEFAULT ''"},
	} {
		if err := s.addColumnIfMissing(c.table, c.col, c.decl); err != nil {
			return err
		}
	}
	// drop rows created before multi-user support (they belong to no account)
	for _, stmt := range []string{
		`DELETE FROM restaurant_rating_submissions WHERE form_id IN (SELECT id FROM restaurant_rating_workflow_forms WHERE user_id = 0)`,
		`DELETE FROM restaurant_rating_evaluation_items WHERE user_id = 0`,
		`DELETE FROM restaurant_rating_workflow_forms WHERE user_id = 0`,
	} {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	for _, idx := range []string{"idx_submissions_view_token", "idx_restaurant_submissions_view_token"} {
		if _, err := s.db.Exec(`DROP INDEX IF EXISTS ` + idx); err != nil {
			return err
		}
	}
	_, err := s.db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_restaurant_rating_submissions_view_token ON restaurant_rating_submissions (view_token)`)
	return err
}

// renameTableIfExists renames oldName to newName when oldName exists and
// newName does not (idempotent; safe to run on every startup).
func (s *Store) renameTableIfExists(oldName, newName string) error {
	if !s.tableExists(oldName) || s.tableExists(newName) {
		return nil
	}
	_, err := s.db.Exec(`ALTER TABLE ` + oldName + ` RENAME TO ` + newName)
	return err
}

func (s *Store) tableExists(name string) bool {
	var found string
	err := s.db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, name).Scan(&found)
	return err == nil
}

func (s *Store) addColumnIfMissing(table, col, decl string) error {
	rows, err := s.db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return err
	}
	existing := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt any
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			rows.Close()
			return err
		}
		existing[name] = true
	}
	rows.Close()
	if existing[col] {
		return nil
	}
	_, err = s.db.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + col + ` ` + decl)
	return err
}

// ---- users ----

func (s *Store) createUser(username, passwordHash string) (User, error) {
	res, err := s.db.Exec(`INSERT INTO users (username, password_hash) VALUES (?, ?)`, username, passwordHash)
	if err != nil {
		return User{}, err
	}
	id, _ := res.LastInsertId()
	var u User
	err = s.db.QueryRow(`SELECT id, username, created_at FROM users WHERE id = ?`, id).
		Scan(&u.ID, &u.Username, &u.CreatedAt)
	return u, err
}

func (s *Store) userByUsername(username string) (User, string, error) {
	var u User
	var hash string
	err := s.db.QueryRow(`SELECT id, username, password_hash, created_at FROM users WHERE username = ?`, username).
		Scan(&u.ID, &u.Username, &hash, &u.CreatedAt)
	return u, hash, err
}

func (s *Store) passwordHashByID(id int64) (string, error) {
	var hash string
	err := s.db.QueryRow(`SELECT password_hash FROM users WHERE id = ?`, id).Scan(&hash)
	return hash, err
}

func (s *Store) updatePassword(id int64, hash string) error {
	res, err := s.db.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, hash, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ---- uploads (stored as BLOB in the database) ----

type Upload struct {
	ID          string
	Filename    string
	ContentType string
	Size        int64
	Data        []byte
	CreatedAt   string
}

func (s *Store) createUpload(u *Upload) error {
	_, err := s.db.Exec(
		`INSERT INTO uploads (id, filename, content_type, size, data) VALUES (?, ?, ?, ?, ?)`,
		u.ID, u.Filename, u.ContentType, u.Size, u.Data)
	return err
}

func (s *Store) getUpload(id string) (*Upload, error) {
	var u Upload
	err := s.db.QueryRow(
		`SELECT id, filename, content_type, size, data, created_at FROM uploads WHERE id = ?`, id).
		Scan(&u.ID, &u.Filename, &u.ContentType, &u.Size, &u.Data, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// ---- evaluation items ----

func (s *Store) listRestaurantRatingItems(userID int64) ([]RestaurantRatingItem, error) {
	rows, err := s.db.Query(`SELECT id, name, description, created_at FROM restaurant_rating_evaluation_items WHERE user_id = ? ORDER BY id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []RestaurantRatingItem{}
	for rows.Next() {
		var it RestaurantRatingItem
		if err := rows.Scan(&it.ID, &it.Name, &it.Description, &it.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func (s *Store) createRestaurantRatingItem(userID int64, name, description string) (RestaurantRatingItem, error) {
	res, err := s.db.Exec(`INSERT INTO restaurant_rating_evaluation_items (user_id, name, description) VALUES (?, ?, ?)`, userID, name, description)
	if err != nil {
		return RestaurantRatingItem{}, err
	}
	id, _ := res.LastInsertId()
	return s.getRestaurantRatingItem(id, userID)
}

func (s *Store) getRestaurantRatingItem(id, userID int64) (RestaurantRatingItem, error) {
	var it RestaurantRatingItem
	err := s.db.QueryRow(`SELECT id, name, description, created_at FROM restaurant_rating_evaluation_items WHERE id = ? AND user_id = ?`, id, userID).
		Scan(&it.ID, &it.Name, &it.Description, &it.CreatedAt)
	return it, err
}

func (s *Store) updateRestaurantRatingItem(userID, id int64, name, description string) error {
	res, err := s.db.Exec(`UPDATE restaurant_rating_evaluation_items SET name = ?, description = ? WHERE id = ? AND user_id = ?`, name, description, id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) deleteRestaurantRatingItem(userID, id int64) error {
	res, err := s.db.Exec(`DELETE FROM restaurant_rating_evaluation_items WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ---- workflow forms ----

func (s *Store) listRestaurantRatingForms(userID int64) ([]RestaurantRatingForm, error) {
	rows, err := s.db.Query(`
		SELECT f.id, f.name, f.share_token, f.created_at, COUNT(fi.item_id)
		FROM restaurant_rating_workflow_forms f
		LEFT JOIN restaurant_rating_form_items fi ON fi.form_id = f.id
		WHERE f.user_id = ?
		GROUP BY f.id
		ORDER BY f.id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	forms := []RestaurantRatingForm{}
	for rows.Next() {
		var f RestaurantRatingForm
		var tok sql.NullString
		if err := rows.Scan(&f.ID, &f.Name, &tok, &f.CreatedAt, &f.ItemCount); err != nil {
			return nil, err
		}
		f.Published = tok.Valid
		f.ShareToken = tok.String
		forms = append(forms, f)
	}
	return forms, rows.Err()
}

// listPublishedRestaurantRatingForms returns forms with a share token, for the public index.
func (s *Store) listPublishedRestaurantRatingForms() ([]RestaurantRatingForm, error) {
	rows, err := s.db.Query(`
		SELECT f.id, f.name, f.share_token, f.created_at, COUNT(fi.item_id)
		FROM restaurant_rating_workflow_forms f
		LEFT JOIN restaurant_rating_form_items fi ON fi.form_id = f.id
		WHERE f.share_token IS NOT NULL
		GROUP BY f.id
		ORDER BY f.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	forms := []RestaurantRatingForm{}
	for rows.Next() {
		var f RestaurantRatingForm
		var tok sql.NullString
		if err := rows.Scan(&f.ID, &f.Name, &tok, &f.CreatedAt, &f.ItemCount); err != nil {
			return nil, err
		}
		f.Published = tok.Valid
		f.ShareToken = tok.String
		forms = append(forms, f)
	}
	return forms, rows.Err()
}

func (s *Store) getRestaurantRatingForm(id, userID int64) (RestaurantRatingForm, error) {
	var f RestaurantRatingForm
	var tok sql.NullString
	err := s.db.QueryRow(`SELECT id, name, share_token, created_at FROM restaurant_rating_workflow_forms WHERE id = ? AND user_id = ?`, id, userID).
		Scan(&f.ID, &f.Name, &tok, &f.CreatedAt)
	if err != nil {
		return RestaurantRatingForm{}, err
	}
	return s.restaurantRatingFormFromScan(f, tok)
}

// getRestaurantRatingFormAny loads a form regardless of owner; used by public result views.
func (s *Store) getRestaurantRatingFormAny(id int64) (RestaurantRatingForm, error) {
	var f RestaurantRatingForm
	var tok sql.NullString
	err := s.db.QueryRow(`SELECT id, name, share_token, created_at FROM restaurant_rating_workflow_forms WHERE id = ?`, id).
		Scan(&f.ID, &f.Name, &tok, &f.CreatedAt)
	if err != nil {
		return RestaurantRatingForm{}, err
	}
	return s.restaurantRatingFormFromScan(f, tok)
}

func (s *Store) restaurantRatingFormFromScan(f RestaurantRatingForm, tok sql.NullString) (RestaurantRatingForm, error) {
	f.Published = tok.Valid
	f.ShareToken = tok.String
	items, err := s.restaurantRatingFormItems(f.ID)
	if err != nil {
		return RestaurantRatingForm{}, err
	}
	f.Items = items
	f.ItemCount = len(items)
	return f, nil
}

func (s *Store) restaurantRatingFormItems(formID int64) ([]RestaurantRatingItem, error) {
	rows, err := s.db.Query(`
		SELECT i.id, i.name, i.description, i.created_at
		FROM restaurant_rating_form_items fi
		JOIN restaurant_rating_evaluation_items i ON i.id = fi.item_id
		WHERE fi.form_id = ?
		ORDER BY fi.position ASC, fi.item_id ASC`, formID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []RestaurantRatingItem{}
	for rows.Next() {
		var it RestaurantRatingItem
		if err := rows.Scan(&it.ID, &it.Name, &it.Description, &it.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func (s *Store) createRestaurantRatingForm(userID int64, name string, itemIDs []int64) (RestaurantRatingForm, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return RestaurantRatingForm{}, err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`INSERT INTO restaurant_rating_workflow_forms (user_id, name) VALUES (?, ?)`, userID, name)
	if err != nil {
		return RestaurantRatingForm{}, err
	}
	id, _ := res.LastInsertId()
	if err := replaceRestaurantRatingFormItems(tx, id, itemIDs); err != nil {
		return RestaurantRatingForm{}, err
	}
	if err := tx.Commit(); err != nil {
		return RestaurantRatingForm{}, err
	}
	return s.getRestaurantRatingForm(id, userID)
}

func (s *Store) updateRestaurantRatingForm(userID, id int64, name string, itemIDs []int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE restaurant_rating_workflow_forms SET name = ? WHERE id = ? AND user_id = ?`, name, id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	if err := replaceRestaurantRatingFormItems(tx, id, itemIDs); err != nil {
		return err
	}
	return tx.Commit()
}

func replaceRestaurantRatingFormItems(tx *sql.Tx, formID int64, itemIDs []int64) error {
	if _, err := tx.Exec(`DELETE FROM restaurant_rating_form_items WHERE form_id = ?`, formID); err != nil {
		return err
	}
	for pos, itemID := range itemIDs {
		if _, err := tx.Exec(`INSERT INTO restaurant_rating_form_items (form_id, item_id, position) VALUES (?, ?, ?)`, formID, itemID, pos); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) deleteRestaurantRatingForm(userID, id int64) error {
	res, err := s.db.Exec(`DELETE FROM restaurant_rating_workflow_forms WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) setRestaurantRatingShareToken(userID, id int64, token *string) error {
	res, err := s.db.Exec(`UPDATE restaurant_rating_workflow_forms SET share_token = ? WHERE id = ? AND user_id = ?`, token, id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) restaurantRatingFormByToken(token string) (RestaurantRatingForm, error) {
	var f RestaurantRatingForm
	var tok sql.NullString
	err := s.db.QueryRow(`SELECT id, name, share_token, created_at FROM restaurant_rating_workflow_forms WHERE share_token = ?`, token).
		Scan(&f.ID, &f.Name, &tok, &f.CreatedAt)
	if err != nil {
		return RestaurantRatingForm{}, err
	}
	f.Published = true
	f.ShareToken = tok.String
	items, err := s.restaurantRatingFormItems(f.ID)
	if err != nil {
		return RestaurantRatingForm{}, err
	}
	f.Items = items
	f.ItemCount = len(items)
	return f, nil
}

func (s *Store) restaurantRatingFormItemIDs(formID int64) (map[int64]bool, error) {
	rows, err := s.db.Query(`SELECT item_id FROM restaurant_rating_form_items WHERE form_id = ?`, formID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids[id] = true
	}
	return ids, rows.Err()
}

// ---- restaurant_rating_submissions ----

type newRestaurantRatingScore struct {
	ItemID   int64
	Score    int
	Evidence []string
}

func (s *Store) createRestaurantRatingSubmission(formID int64, restaurant, evaluator string, scores []newRestaurantRatingScore) (RestaurantRatingSubmission, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return RestaurantRatingSubmission{}, err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`INSERT INTO restaurant_rating_submissions (form_id, view_token, restaurant, evaluator) VALUES (?, ?, ?, ?)`,
		formID, randomHex(16), restaurant, evaluator)
	if err != nil {
		return RestaurantRatingSubmission{}, err
	}
	subID, _ := res.LastInsertId()
	for _, sc := range scores {
		if _, err := tx.Exec(`INSERT INTO restaurant_rating_submission_scores (submission_id, item_id, score, evidence) VALUES (?, ?, ?, ?)`,
			subID, sc.ItemID, sc.Score, strings.Join(sc.Evidence, "\n")); err != nil {
			return RestaurantRatingSubmission{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return RestaurantRatingSubmission{}, err
	}
	return s.getRestaurantRatingSubmission(subID)
}

func (s *Store) getRestaurantRatingSubmission(id int64) (RestaurantRatingSubmission, error) {
	var sub RestaurantRatingSubmission
	if err := s.db.QueryRow(`SELECT id, form_id, view_token, restaurant, evaluator, created_at FROM restaurant_rating_submissions WHERE id = ?`, id).
		Scan(&sub.ID, &sub.FormID, &sub.ViewToken, &sub.Restaurant, &sub.Evaluator, &sub.CreatedAt); err != nil {
		return RestaurantRatingSubmission{}, err
	}
	scores, err := s.restaurantRatingSubmissionScores(id)
	if err != nil {
		return RestaurantRatingSubmission{}, err
	}
	sub.Scores = scores
	sub.TotalScore, sub.MaxScore, sub.PassedCount, sub.TotalCount = computeRestaurantRatingStats(scores)
	return sub, nil
}

func (s *Store) restaurantRatingSubmissionByViewToken(token string) (RestaurantRatingForm, RestaurantRatingSubmission, error) {
	var sub RestaurantRatingSubmission
	if err := s.db.QueryRow(`SELECT id, form_id, view_token, restaurant, evaluator, created_at FROM restaurant_rating_submissions WHERE view_token = ?`, token).
		Scan(&sub.ID, &sub.FormID, &sub.ViewToken, &sub.Restaurant, &sub.Evaluator, &sub.CreatedAt); err != nil {
		return RestaurantRatingForm{}, RestaurantRatingSubmission{}, err
	}
	scores, err := s.restaurantRatingSubmissionScores(sub.ID)
	if err != nil {
		return RestaurantRatingForm{}, RestaurantRatingSubmission{}, err
	}
	sub.Scores = scores
	sub.TotalScore, sub.MaxScore, sub.PassedCount, sub.TotalCount = computeRestaurantRatingStats(scores)
	form, err := s.getRestaurantRatingFormAny(sub.FormID)
	if err != nil {
		return RestaurantRatingForm{}, RestaurantRatingSubmission{}, err
	}
	return form, sub, nil
}

func (s *Store) restaurantRatingSubmissionScores(subID int64) ([]RestaurantRatingScore, error) {
	rows, err := s.db.Query(`
		SELECT ss.item_id, i.name, ss.score, ss.evidence
		FROM restaurant_rating_submission_scores ss
		JOIN restaurant_rating_evaluation_items i ON i.id = ss.item_id
		WHERE ss.submission_id = ?
		ORDER BY ss.id ASC`, subID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	scores := []RestaurantRatingScore{}
	for rows.Next() {
		var sc RestaurantRatingScore
		var ev string
		if err := rows.Scan(&sc.ItemID, &sc.ItemName, &sc.Score, &ev); err != nil {
			return nil, err
		}
		sc.Passed = sc.Score >= restaurantRatingPassScore
		sc.Evidence = []string{}
		for _, u := range strings.Split(ev, "\n") {
			if u != "" {
				sc.Evidence = append(sc.Evidence, u)
			}
		}
		scores = append(scores, sc)
	}
	return scores, rows.Err()
}

func (s *Store) listRestaurantRatingSubmissions(formID int64) ([]RestaurantRatingSubmission, error) {
	rows, err := s.db.Query(`SELECT id, form_id, view_token, restaurant, evaluator, created_at FROM restaurant_rating_submissions WHERE form_id = ? ORDER BY id DESC`, formID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	subs := []RestaurantRatingSubmission{}
	for rows.Next() {
		var sub RestaurantRatingSubmission
		if err := rows.Scan(&sub.ID, &sub.FormID, &sub.ViewToken, &sub.Restaurant, &sub.Evaluator, &sub.CreatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	for i := range subs {
		scores, err := s.restaurantRatingSubmissionScores(subs[i].ID)
		if err != nil {
			return nil, err
		}
		subs[i].Scores = scores
		subs[i].TotalScore, subs[i].MaxScore, subs[i].PassedCount, subs[i].TotalCount = computeRestaurantRatingStats(scores)
	}
	return subs, nil
}

func (s *Store) listAllRestaurantRatingSubmissions(userID int64) ([]RestaurantRatingSubmission, error) {
	rows, err := s.db.Query(`
		SELECT s.id, s.form_id, s.view_token, s.restaurant, s.evaluator, s.created_at, f.name
		FROM restaurant_rating_submissions s
		JOIN restaurant_rating_workflow_forms f ON f.id = s.form_id
		WHERE f.user_id = ?
		ORDER BY s.id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	subs := []RestaurantRatingSubmission{}
	for rows.Next() {
		var sub RestaurantRatingSubmission
		if err := rows.Scan(&sub.ID, &sub.FormID, &sub.ViewToken, &sub.Restaurant, &sub.Evaluator, &sub.CreatedAt, &sub.FormName); err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range subs {
		scores, err := s.restaurantRatingSubmissionScores(subs[i].ID)
		if err != nil {
			return nil, err
		}
		subs[i].Scores = scores
		subs[i].TotalScore, subs[i].MaxScore, subs[i].PassedCount, subs[i].TotalCount = computeRestaurantRatingStats(scores)
	}
	return subs, nil
}
