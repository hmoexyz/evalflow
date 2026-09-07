package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	store *Store
	auth  *authStore
}

type ctxKey int

const userKey ctxKey = 0

// currentUser returns the id of the authenticated user (set by requireAuth).
func currentUser(r *http.Request) int64 {
	return r.Context().Value(userKey).(int64)
}

const maxUploadBytes = 20 << 20 // 20MB

var allowedExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
	".bmp": true, ".pdf": true, ".doc": true, ".docx": true, ".xls": true,
	".xlsx": true, ".ppt": true, ".pptx": true, ".txt": true, ".mp4": true, ".mov": true,
}

func (s *Server) routes() *http.ServeMux {
	mux := http.NewServeMux()

	// public
	mux.HandleFunc("POST /api/register", s.handleRegister)
	mux.HandleFunc("POST /api/login", s.handleLogin)
	mux.HandleFunc("GET /api/share", s.handleListPublishedForms)
	mux.Handle("POST /api/upload", s.limitBody(s.handleUpload))
	mux.HandleFunc("GET /uploads/{name}", s.handleGetUpload)
	mux.HandleFunc("GET /api/share/{token}", s.handleGetSharedForm)
	mux.Handle("POST /api/share/{token}/submissions", s.limitBody(s.handleSubmit))
	mux.HandleFunc("GET /api/results/{token}", s.handleGetResult)

	// admin
	mux.Handle("POST /api/password", s.requireAuth(s.handleChangePassword))
	mux.Handle("GET /api/items", s.requireAuth(s.handleListItems))
	mux.Handle("POST /api/items", s.requireAuth(s.handleCreateItem))
	mux.Handle("PUT /api/items/{id}", s.requireAuth(s.handleUpdateItem))
	mux.Handle("DELETE /api/items/{id}", s.requireAuth(s.handleDeleteItem))

	mux.Handle("GET /api/forms", s.requireAuth(s.handleListForms))
	mux.Handle("POST /api/forms", s.requireAuth(s.handleCreateForm))
	mux.Handle("GET /api/forms/{id}", s.requireAuth(s.handleGetForm))
	mux.Handle("PUT /api/forms/{id}", s.requireAuth(s.handleUpdateForm))
	mux.Handle("DELETE /api/forms/{id}", s.requireAuth(s.handleDeleteForm))
	mux.Handle("POST /api/forms/{id}/publish", s.requireAuth(s.handlePublishForm))
	mux.Handle("POST /api/forms/{id}/unpublish", s.requireAuth(s.handleUnpublishForm))
	mux.Handle("GET /api/forms/{id}/submissions", s.requireAuth(s.handleListSubmissions))
	mux.Handle("GET /api/submissions", s.requireAuth(s.handleListAllSubmissions))

	return mux
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) limitBody(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
		next(w, r)
	})
}

func (s *Server) requireAuth(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok {
			writeErr(w, http.StatusUnauthorized, "未登录或登录已过期")
			return
		}
		userID, ok := s.auth.lookup(tok)
		if !ok {
			writeErr(w, http.StatusUnauthorized, "未登录或登录已过期")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), userKey, userID)))
	})
}

// notFound maps a store miss to 404; other errors stay nil so callers can 500.
func isNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

func pathID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

// ---- auth ----

func validUsername(u string) bool {
	if n := len(u); n < 2 || n > 32 {
		return false
	}
	return !strings.ContainsAny(u, " \t\r\n")
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if !validUsername(req.Username) {
		writeErr(w, http.StatusBadRequest, "用户名需为 2~32 个字符且不能包含空格")
		return
	}
	if len(req.Password) < 6 || len(req.Password) > 72 {
		writeErr(w, http.StatusBadRequest, "密码长度需为 6~72 个字符")
		return
	}
	if _, err := s.store.createUser(req.Username, passwordHash(req.Password)); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			writeErr(w, http.StatusConflict, "用户名已存在")
			return
		}
		writeErr(w, http.StatusInternalServerError, "注册失败")
		return
	}
	u, _, err := s.store.userByUsername(req.Username)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "注册失败")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"token": s.auth.issue(u.ID), "username": u.Username})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	u, hash, err := s.store.userByUsername(strings.TrimSpace(req.Username))
	if err != nil || !passwordVerify(req.Password, hash) {
		writeErr(w, http.StatusUnauthorized, "用户名或密码错误")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": s.auth.issue(u.ID), "username": u.Username})
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if len(req.NewPassword) < 6 || len(req.NewPassword) > 72 {
		writeErr(w, http.StatusBadRequest, "新密码长度需为 6~72 个字符")
		return
	}
	userID := currentUser(r)
	hash, err := s.store.passwordHashByID(userID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "读取账号失败")
		return
	}
	if !passwordVerify(req.OldPassword, hash) {
		writeErr(w, http.StatusBadRequest, "当前密码错误")
		return
	}
	if err := s.store.updatePassword(userID, passwordHash(req.NewPassword)); err != nil {
		writeErr(w, http.StatusInternalServerError, "修改密码失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ---- items ----

func (s *Server) handleListItems(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.listItems(currentUser(r))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "读取评估项失败")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleCreateItem(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeErr(w, http.StatusBadRequest, "名称不能为空")
		return
	}
	item, err := s.store.createItem(currentUser(r), req.Name, strings.TrimSpace(req.Description))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "创建评估项失败")
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handleUpdateItem(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeErr(w, http.StatusBadRequest, "名称不能为空")
		return
	}
	if err := s.store.updateItem(currentUser(r), id, req.Name, strings.TrimSpace(req.Description)); err != nil {
		if isNotFound(err) {
			writeErr(w, http.StatusNotFound, "评估项不存在")
			return
		}
		writeErr(w, http.StatusInternalServerError, "更新评估项失败")
		return
	}
	item, err := s.store.getItem(id, currentUser(r))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "读取评估项失败")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleDeleteItem(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	if err := s.store.deleteItem(currentUser(r), id); err != nil {
		if isNotFound(err) {
			writeErr(w, http.StatusNotFound, "评估项不存在")
			return
		}
		writeErr(w, http.StatusInternalServerError, "删除评估项失败")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- forms ----

func (s *Server) handleListForms(w http.ResponseWriter, r *http.Request) {
	forms, err := s.store.listForms(currentUser(r))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "读取流程表失败")
		return
	}
	writeJSON(w, http.StatusOK, forms)
}

// ownedItemIDs returns the set of item ids belonging to the given user.
func (s *Server) ownedItemIDs(userID int64) (map[int64]bool, error) {
	items, err := s.store.listItems(userID)
	if err != nil {
		return nil, err
	}
	ids := make(map[int64]bool, len(items))
	for _, it := range items {
		ids[it.ID] = true
	}
	return ids, nil
}

func (s *Server) handleCreateForm(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string  `json:"name"`
		ItemIDs []int64 `json:"item_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeErr(w, http.StatusBadRequest, "名称不能为空")
		return
	}
	if len(req.ItemIDs) == 0 {
		writeErr(w, http.StatusBadRequest, "流程表至少需要包含一个评估项")
		return
	}
	userID := currentUser(r)
	owned, err := s.ownedItemIDs(userID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "读取评估项失败")
		return
	}
	for _, id := range req.ItemIDs {
		if !owned[id] {
			writeErr(w, http.StatusBadRequest, "流程表包含不属于你的评估项")
			return
		}
	}
	form, err := s.store.createForm(userID, req.Name, req.ItemIDs)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "创建流程表失败")
		return
	}
	writeJSON(w, http.StatusCreated, form)
}

func (s *Server) handleGetForm(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	form, err := s.store.getForm(id, currentUser(r))
	if err != nil {
		if isNotFound(err) {
			writeErr(w, http.StatusNotFound, "流程表不存在")
			return
		}
		writeErr(w, http.StatusInternalServerError, "读取流程表失败")
		return
	}
	writeJSON(w, http.StatusOK, form)
}

func (s *Server) handleUpdateForm(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	var req struct {
		Name    string  `json:"name"`
		ItemIDs []int64 `json:"item_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeErr(w, http.StatusBadRequest, "名称不能为空")
		return
	}
	if len(req.ItemIDs) == 0 {
		writeErr(w, http.StatusBadRequest, "流程表至少需要包含一个评估项")
		return
	}
	userID := currentUser(r)
	owned, err := s.ownedItemIDs(userID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "读取评估项失败")
		return
	}
	for _, iid := range req.ItemIDs {
		if !owned[iid] {
			writeErr(w, http.StatusBadRequest, "流程表包含不属于你的评估项")
			return
		}
	}
	if err := s.store.updateForm(userID, id, req.Name, req.ItemIDs); err != nil {
		if isNotFound(err) {
			writeErr(w, http.StatusNotFound, "流程表不存在")
			return
		}
		writeErr(w, http.StatusInternalServerError, "更新流程表失败")
		return
	}
	form, err := s.store.getForm(id, userID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "读取流程表失败")
		return
	}
	writeJSON(w, http.StatusOK, form)
}

func (s *Server) handleDeleteForm(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	if err := s.store.deleteForm(currentUser(r), id); err != nil {
		if isNotFound(err) {
			writeErr(w, http.StatusNotFound, "流程表不存在")
			return
		}
		writeErr(w, http.StatusInternalServerError, "删除流程表失败")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePublishForm(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	tok := randomHex(16)
	if err := s.store.setShareToken(currentUser(r), id, &tok); err != nil {
		if isNotFound(err) {
			writeErr(w, http.StatusNotFound, "流程表不存在")
			return
		}
		writeErr(w, http.StatusInternalServerError, "发布流程表失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": tok, "url": "/share/" + tok})
}

func (s *Server) handleUnpublishForm(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	if err := s.store.setShareToken(currentUser(r), id, nil); err != nil {
		if isNotFound(err) {
			writeErr(w, http.StatusNotFound, "流程表不存在")
			return
		}
		writeErr(w, http.StatusInternalServerError, "取消发布失败")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- submissions ----

func (s *Server) handleListAllSubmissions(w http.ResponseWriter, r *http.Request) {
	subs, err := s.store.listAllSubmissions(currentUser(r))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "读取评测结果失败")
		return
	}
	writeJSON(w, http.StatusOK, subs)
}

func (s *Server) handleListSubmissions(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	if _, err := s.store.getForm(id, currentUser(r)); err != nil {
		if isNotFound(err) {
			writeErr(w, http.StatusNotFound, "流程表不存在")
			return
		}
		writeErr(w, http.StatusInternalServerError, "读取流程表失败")
		return
	}
	subs, err := s.store.listSubmissions(id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "读取提交记录失败")
		return
	}
	writeJSON(w, http.StatusOK, subs)
}

// ---- public share ----

func (s *Server) handleListPublishedForms(w http.ResponseWriter, r *http.Request) {
	forms, err := s.store.listPublishedForms()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "读取流程表失败")
		return
	}
	writeJSON(w, http.StatusOK, forms)
}

func (s *Server) handleGetSharedForm(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	form, err := s.store.formByToken(token)
	if err != nil {
		writeErr(w, http.StatusNotFound, "流程表不存在或未发布")
		return
	}
	writeJSON(w, http.StatusOK, form)
}

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	form, err := s.store.formByToken(token)
	if err != nil {
		writeErr(w, http.StatusNotFound, "流程表不存在或未发布")
		return
	}
	validItems, err := s.store.formItemIDs(form.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "读取流程表失败")
		return
	}

	var req struct {
		Restaurant string `json:"restaurant"`
		Evaluator  string `json:"evaluator"`
		Scores     []struct {
			ItemID   int64    `json:"item_id"`
			Score    int      `json:"score"`
			Evidence []string `json:"evidence"`
		} `json:"scores"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	req.Restaurant = strings.TrimSpace(req.Restaurant)
	req.Evaluator = strings.TrimSpace(req.Evaluator)
	if req.Restaurant == "" {
		writeErr(w, http.StatusBadRequest, "请填写所在餐厅")
		return
	}
	if req.Evaluator == "" {
		writeErr(w, http.StatusBadRequest, "请填写测评人")
		return
	}
	if len(req.Scores) == 0 {
		writeErr(w, http.StatusBadRequest, "至少需要一个评分")
		return
	}
	if len(req.Scores) != len(validItems) {
		writeErr(w, http.StatusBadRequest, "请对每个评估项进行评分")
		return
	}

	seen := make(map[int64]bool, len(req.Scores))
	scores := make([]newScore, 0, len(req.Scores))
	for _, sc := range req.Scores {
		if !validItems[sc.ItemID] {
			writeErr(w, http.StatusBadRequest, "评分中包含不属于该流程表的评估项")
			return
		}
		if seen[sc.ItemID] {
			writeErr(w, http.StatusBadRequest, "存在重复的评估项评分")
			return
		}
		seen[sc.ItemID] = true
		if sc.Score < 0 || sc.Score > 10 {
			writeErr(w, http.StatusBadRequest, "评分必须在 0~10 之间")
			return
		}
		var evs []string
		for _, ev := range sc.Evidence {
			ev = strings.TrimSpace(ev)
			if ev == "" {
				continue
			}
			if !strings.HasPrefix(ev, "/uploads/") || strings.Contains(ev, "..") {
				writeErr(w, http.StatusBadRequest, "证据文件路径无效")
				return
			}
			evs = append(evs, ev)
		}
		scores = append(scores, newScore{ItemID: sc.ItemID, Score: sc.Score, Evidence: evs})
	}

	sub, err := s.store.createSubmission(form.ID, req.Restaurant, req.Evaluator, scores)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "保存评分失败")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"submission": sub})
}

// ---- public result view ----

func (s *Server) handleGetResult(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	form, sub, err := s.store.submissionByViewToken(token)
	if err != nil {
		writeErr(w, http.StatusNotFound, "结果不存在")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"form": form, "submission": sub})
}

// ---- upload ----

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("file")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "未找到上传文件")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedExts[ext] {
		writeErr(w, http.StatusBadRequest, "不支持的文件类型")
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "读取文件失败")
		return
	}
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(ext)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}

	// 文件以 BLOB 形式直接存入数据库，URL 保留扩展名以兼容前端预览逻辑
	id := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), randomHex(4), ext)
	u := &Upload{
		ID:          id,
		Filename:    filepath.Base(header.Filename),
		ContentType: contentType,
		Size:        int64(len(data)),
		Data:        data,
	}
	if err := s.store.createUpload(u); err != nil {
		writeErr(w, http.StatusInternalServerError, "保存文件失败")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"url": "/uploads/" + id})
}

func (s *Server) handleGetUpload(w http.ResponseWriter, r *http.Request) {
	u, err := s.store.getUpload(r.PathValue("name"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "文件不存在")
		return
	}
	disposition := mime.FormatMediaType("inline", map[string]string{"filename": u.Filename})
	w.Header().Set("Content-Disposition", disposition)
	w.Header().Set("Content-Type", u.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(u.Size, 10))
	w.Write(u.Data)
}
