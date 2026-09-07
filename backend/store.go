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
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL COLLATE NOCASE UNIQUE,
			password_hash TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
		)`,
		`CREATE TABLE IF NOT EXISTS evaluation_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL DEFAULT 0,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
		)`,
		`CREATE TABLE IF NOT EXISTS workflow_forms (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL DEFAULT 0,
			name TEXT NOT NULL,
			share_token TEXT UNIQUE,
			created_at TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
		)`,
		`CREATE TABLE IF NOT EXISTS form_items (
			form_id INTEGER NOT NULL REFERENCES workflow_forms(id) ON DELETE CASCADE,
			item_id INTEGER NOT NULL REFERENCES evaluation_items(id) ON DELETE CASCADE,
			position INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (form_id, item_id)
		)`,
		`CREATE TABLE IF NOT EXISTS submissions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			form_id INTEGER NOT NULL REFERENCES workflow_forms(id) ON DELETE CASCADE,
			view_token TEXT,
			restaurant TEXT NOT NULL DEFAULT '',
			evaluator TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
		)`,
		`CREATE TABLE IF NOT EXISTS submission_scores (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			submission_id INTEGER NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
			item_id INTEGER NOT NULL REFERENCES evaluation_items(id),
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
		{"evaluation_items", "user_id", "INTEGER NOT NULL DEFAULT 0"},
		{"workflow_forms", "user_id", "INTEGER NOT NULL DEFAULT 0"},
		{"submissions", "view_token", "TEXT"},
		{"submissions", "restaurant", "TEXT NOT NULL DEFAULT ''"},
		{"submissions", "evaluator", "TEXT NOT NULL DEFAULT ''"},
	} {
		if err := s.addColumnIfMissing(c.table, c.col, c.decl); err != nil {
			return err
		}
	}
	// drop rows created before multi-user support (they belong to no account)
	for _, stmt := range []string{
		`DELETE FROM submissions WHERE form_id IN (SELECT id FROM workflow_forms WHERE user_id = 0)`,
		`DELETE FROM evaluation_items WHERE user_id = 0`,
		`DELETE FROM workflow_forms WHERE user_id = 0`,
	} {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	_, err := s.db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_submissions_view_token ON submissions (view_token)`)
	return err
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

func (s *Store) listItems(userID int64) ([]Item, error) {
	rows, err := s.db.Query(`SELECT id, name, description, created_at FROM evaluation_items WHERE user_id = ? ORDER BY id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Item{}
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.Name, &it.Description, &it.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func (s *Store) createItem(userID int64, name, description string) (Item, error) {
	res, err := s.db.Exec(`INSERT INTO evaluation_items (user_id, name, description) VALUES (?, ?, ?)`, userID, name, description)
	if err != nil {
		return Item{}, err
	}
	id, _ := res.LastInsertId()
	return s.getItem(id, userID)
}

func (s *Store) getItem(id, userID int64) (Item, error) {
	var it Item
	err := s.db.QueryRow(`SELECT id, name, description, created_at FROM evaluation_items WHERE id = ? AND user_id = ?`, id, userID).
		Scan(&it.ID, &it.Name, &it.Description, &it.CreatedAt)
	return it, err
}

func (s *Store) updateItem(userID, id int64, name, description string) error {
	res, err := s.db.Exec(`UPDATE evaluation_items SET name = ?, description = ? WHERE id = ? AND user_id = ?`, name, description, id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) deleteItem(userID, id int64) error {
	res, err := s.db.Exec(`DELETE FROM evaluation_items WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ---- workflow forms ----

func (s *Store) listForms(userID int64) ([]Form, error) {
	rows, err := s.db.Query(`
		SELECT f.id, f.name, f.share_token, f.created_at, COUNT(fi.item_id)
		FROM workflow_forms f
		LEFT JOIN form_items fi ON fi.form_id = f.id
		WHERE f.user_id = ?
		GROUP BY f.id
		ORDER BY f.id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	forms := []Form{}
	for rows.Next() {
		var f Form
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

// listPublishedForms returns forms with a share token, for the public index.
func (s *Store) listPublishedForms() ([]Form, error) {
	rows, err := s.db.Query(`
		SELECT f.id, f.name, f.share_token, f.created_at, COUNT(fi.item_id)
		FROM workflow_forms f
		LEFT JOIN form_items fi ON fi.form_id = f.id
		WHERE f.share_token IS NOT NULL
		GROUP BY f.id
		ORDER BY f.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	forms := []Form{}
	for rows.Next() {
		var f Form
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

func (s *Store) getForm(id, userID int64) (Form, error) {
	var f Form
	var tok sql.NullString
	err := s.db.QueryRow(`SELECT id, name, share_token, created_at FROM workflow_forms WHERE id = ? AND user_id = ?`, id, userID).
		Scan(&f.ID, &f.Name, &tok, &f.CreatedAt)
	if err != nil {
		return Form{}, err
	}
	return s.formFromScan(f, tok)
}

// getFormAny loads a form regardless of owner; used by public result views.
func (s *Store) getFormAny(id int64) (Form, error) {
	var f Form
	var tok sql.NullString
	err := s.db.QueryRow(`SELECT id, name, share_token, created_at FROM workflow_forms WHERE id = ?`, id).
		Scan(&f.ID, &f.Name, &tok, &f.CreatedAt)
	if err != nil {
		return Form{}, err
	}
	return s.formFromScan(f, tok)
}

func (s *Store) formFromScan(f Form, tok sql.NullString) (Form, error) {
	f.Published = tok.Valid
	f.ShareToken = tok.String
	items, err := s.formItems(f.ID)
	if err != nil {
		return Form{}, err
	}
	f.Items = items
	f.ItemCount = len(items)
	return f, nil
}

func (s *Store) formItems(formID int64) ([]Item, error) {
	rows, err := s.db.Query(`
		SELECT i.id, i.name, i.description, i.created_at
		FROM form_items fi
		JOIN evaluation_items i ON i.id = fi.item_id
		WHERE fi.form_id = ?
		ORDER BY fi.position ASC, fi.item_id ASC`, formID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Item{}
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.Name, &it.Description, &it.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func (s *Store) createForm(userID int64, name string, itemIDs []int64) (Form, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Form{}, err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`INSERT INTO workflow_forms (user_id, name) VALUES (?, ?)`, userID, name)
	if err != nil {
		return Form{}, err
	}
	id, _ := res.LastInsertId()
	if err := replaceFormItems(tx, id, itemIDs); err != nil {
		return Form{}, err
	}
	if err := tx.Commit(); err != nil {
		return Form{}, err
	}
	return s.getForm(id, userID)
}

func (s *Store) updateForm(userID, id int64, name string, itemIDs []int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE workflow_forms SET name = ? WHERE id = ? AND user_id = ?`, name, id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	if err := replaceFormItems(tx, id, itemIDs); err != nil {
		return err
	}
	return tx.Commit()
}

func replaceFormItems(tx *sql.Tx, formID int64, itemIDs []int64) error {
	if _, err := tx.Exec(`DELETE FROM form_items WHERE form_id = ?`, formID); err != nil {
		return err
	}
	for pos, itemID := range itemIDs {
		if _, err := tx.Exec(`INSERT INTO form_items (form_id, item_id, position) VALUES (?, ?, ?)`, formID, itemID, pos); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) deleteForm(userID, id int64) error {
	res, err := s.db.Exec(`DELETE FROM workflow_forms WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) setShareToken(userID, id int64, token *string) error {
	res, err := s.db.Exec(`UPDATE workflow_forms SET share_token = ? WHERE id = ? AND user_id = ?`, token, id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) formByToken(token string) (Form, error) {
	var f Form
	var tok sql.NullString
	err := s.db.QueryRow(`SELECT id, name, share_token, created_at FROM workflow_forms WHERE share_token = ?`, token).
		Scan(&f.ID, &f.Name, &tok, &f.CreatedAt)
	if err != nil {
		return Form{}, err
	}
	f.Published = true
	f.ShareToken = tok.String
	items, err := s.formItems(f.ID)
	if err != nil {
		return Form{}, err
	}
	f.Items = items
	f.ItemCount = len(items)
	return f, nil
}

func (s *Store) formItemIDs(formID int64) (map[int64]bool, error) {
	rows, err := s.db.Query(`SELECT item_id FROM form_items WHERE form_id = ?`, formID)
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

// ---- submissions ----

type newScore struct {
	ItemID   int64
	Score    int
	Evidence []string
}

func (s *Store) createSubmission(formID int64, restaurant, evaluator string, scores []newScore) (Submission, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Submission{}, err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`INSERT INTO submissions (form_id, view_token, restaurant, evaluator) VALUES (?, ?, ?, ?)`,
		formID, randomHex(16), restaurant, evaluator)
	if err != nil {
		return Submission{}, err
	}
	subID, _ := res.LastInsertId()
	for _, sc := range scores {
		if _, err := tx.Exec(`INSERT INTO submission_scores (submission_id, item_id, score, evidence) VALUES (?, ?, ?, ?)`,
			subID, sc.ItemID, sc.Score, strings.Join(sc.Evidence, "\n")); err != nil {
			return Submission{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Submission{}, err
	}
	return s.getSubmission(subID)
}

func (s *Store) getSubmission(id int64) (Submission, error) {
	var sub Submission
	if err := s.db.QueryRow(`SELECT id, form_id, view_token, restaurant, evaluator, created_at FROM submissions WHERE id = ?`, id).
		Scan(&sub.ID, &sub.FormID, &sub.ViewToken, &sub.Restaurant, &sub.Evaluator, &sub.CreatedAt); err != nil {
		return Submission{}, err
	}
	scores, err := s.submissionScores(id)
	if err != nil {
		return Submission{}, err
	}
	sub.Scores = scores
	sub.TotalScore, sub.MaxScore, sub.PassedCount, sub.TotalCount = computeStats(scores)
	return sub, nil
}

func (s *Store) submissionByViewToken(token string) (Form, Submission, error) {
	var sub Submission
	if err := s.db.QueryRow(`SELECT id, form_id, view_token, restaurant, evaluator, created_at FROM submissions WHERE view_token = ?`, token).
		Scan(&sub.ID, &sub.FormID, &sub.ViewToken, &sub.Restaurant, &sub.Evaluator, &sub.CreatedAt); err != nil {
		return Form{}, Submission{}, err
	}
	scores, err := s.submissionScores(sub.ID)
	if err != nil {
		return Form{}, Submission{}, err
	}
	sub.Scores = scores
	sub.TotalScore, sub.MaxScore, sub.PassedCount, sub.TotalCount = computeStats(scores)
	form, err := s.getFormAny(sub.FormID)
	if err != nil {
		return Form{}, Submission{}, err
	}
	return form, sub, nil
}

func (s *Store) submissionScores(subID int64) ([]Score, error) {
	rows, err := s.db.Query(`
		SELECT ss.item_id, i.name, ss.score, ss.evidence
		FROM submission_scores ss
		JOIN evaluation_items i ON i.id = ss.item_id
		WHERE ss.submission_id = ?
		ORDER BY ss.id ASC`, subID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	scores := []Score{}
	for rows.Next() {
		var sc Score
		var ev string
		if err := rows.Scan(&sc.ItemID, &sc.ItemName, &sc.Score, &ev); err != nil {
			return nil, err
		}
		sc.Passed = sc.Score >= passScore
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

func (s *Store) listSubmissions(formID int64) ([]Submission, error) {
	rows, err := s.db.Query(`SELECT id, form_id, view_token, restaurant, evaluator, created_at FROM submissions WHERE form_id = ? ORDER BY id DESC`, formID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	subs := []Submission{}
	for rows.Next() {
		var sub Submission
		if err := rows.Scan(&sub.ID, &sub.FormID, &sub.ViewToken, &sub.Restaurant, &sub.Evaluator, &sub.CreatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	for i := range subs {
		scores, err := s.submissionScores(subs[i].ID)
		if err != nil {
			return nil, err
		}
		subs[i].Scores = scores
		subs[i].TotalScore, subs[i].MaxScore, subs[i].PassedCount, subs[i].TotalCount = computeStats(scores)
	}
	return subs, nil
}

func (s *Store) listAllSubmissions(userID int64) ([]Submission, error) {
	rows, err := s.db.Query(`
		SELECT s.id, s.form_id, s.view_token, s.restaurant, s.evaluator, s.created_at, f.name
		FROM submissions s
		JOIN workflow_forms f ON f.id = s.form_id
		WHERE f.user_id = ?
		ORDER BY s.id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	subs := []Submission{}
	for rows.Next() {
		var sub Submission
		if err := rows.Scan(&sub.ID, &sub.FormID, &sub.ViewToken, &sub.Restaurant, &sub.Evaluator, &sub.CreatedAt, &sub.FormName); err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range subs {
		scores, err := s.submissionScores(subs[i].ID)
		if err != nil {
			return nil, err
		}
		subs[i].Scores = scores
		subs[i].TotalScore, subs[i].MaxScore, subs[i].PassedCount, subs[i].TotalCount = computeStats(scores)
	}
	return subs, nil
}
