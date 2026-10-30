package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Account struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Username string `json:"username"`
	Kind     string `json:"kind"`
	Enabled  bool   `json:"enabled"`
}
type Reel struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Caption   string    `json:"caption"`
	AccountID string    `json:"accountId"`
	Draft     string    `json:"draft"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}
type State struct {
	Accounts []Account `json:"accounts"`
	Reels    []Reel    `json:"reels"`
	Paused   bool      `json:"paused"`
	LastTick time.Time `json:"lastTick"`
}
type App struct {
	mu    sync.Mutex
	state State
	path  string
}

func newApp(path string) *App {
	a := &App{path: path}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &a.state)
	}
	if a.state.Accounts == nil { a.state.Accounts = []Account{} }
	if a.state.Reels == nil { a.state.Reels = []Reel{} }
	return a
}
func (a *App) saveLocked() {
	b, err := json.MarshalIndent(a.state, "", "  ")
	if err != nil { log.Printf("state encode: %v", err); return }
	tmp := a.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err != nil { log.Printf("state write: %v", err); return }
	if err := os.Rename(tmp, a.path); err != nil { log.Printf("state rename: %v", err) }
}
func id() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }

func makeDraft(caption, tone, invite string) string {
	caption = strings.TrimSpace(caption)
	if caption == "" { caption = "this Reel" }
	lower := strings.ToLower(caption)
	opening := "This is a fun perspective"
	switch {
	case strings.Contains(lower, "travel") || strings.Contains(lower, "trip") || strings.Contains(lower, "place"):
		opening = "This place looks worth exploring"
	case strings.Contains(lower, "food") || strings.Contains(lower, "recipe") || strings.Contains(lower, "cook"):
		opening = "That looks delicious"
	case strings.Contains(lower, "music") || strings.Contains(lower, "song") || strings.Contains(lower, "dance"):
		opening = "The vibe here is great"
	case strings.Contains(lower, "learn") || strings.Contains(lower, "tip") || strings.Contains(lower, "how to"):
		opening = "Useful tip, nicely shared"
	case strings.Contains(lower, "funny") || strings.Contains(lower, "😂") || strings.Contains(lower, "comedy"):
		opening = "This made me smile"
	}
	var s string
	switch tone {
	case "friendly":
		s = opening + " — thanks for sharing!"
	case "curious":
		s = opening + ". What inspired this?"
	case "short":
		s = opening + " 🙌"
	case "hinglish":
		s = opening + " — kaafi achha share kiya!"
	default:
		s = opening + " — really enjoyed this."
	}
	if invite != "" { s += "\n\n" + strings.TrimSpace(invite) }
	return s
}
func (a *App) notify(msg string) {
	token, chat := os.Getenv("TELEGRAM_BOT_TOKEN"), os.Getenv("TELEGRAM_CHAT_ID")
	if token == "" || chat == "" { return }
	body, _ := json.Marshal(map[string]string{"chat_id": chat, "text": msg})
	req, _ := http.NewRequest(http.MethodPost, "https://api.telegram.org/bot"+token+"/sendMessage", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil { log.Printf("telegram notify: %v", err); return }
	defer resp.Body.Close()
}
func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" { http.NotFound(w,r); return }
		http.ServeFile(w,r,"web/index.html")
	})
	mux.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		a.mu.Lock(); defer a.mu.Unlock()
		w.Header().Set("Content-Type","application/json")
		_ = json.NewEncoder(w).Encode(a.state)
	})
	mux.HandleFunc("/api/accounts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost { http.Error(w,"POST required",405); return }
		var in Account
		if err:=json.NewDecoder(r.Body).Decode(&in); err!=nil { http.Error(w,"Invalid JSON",400); return }
		in.Username=strings.TrimSpace(strings.TrimPrefix(in.Username,"@"))
		in.Label=strings.TrimSpace(in.Label)
		if in.Username=="" || in.Label=="" { http.Error(w,"Label and username are required",400); return }
		if in.Kind=="" { in.Kind="Creator / Business (authorized)" }
		in.ID=id(); in.Enabled=true
		a.mu.Lock(); a.state.Accounts=append(a.state.Accounts,in); a.saveLocked(); a.mu.Unlock()
		a.notify("ReelMate: account profile added: @"+in.Username+" (profile only; no Instagram login stored).")
		w.Header().Set("Content-Type","application/json"); w.WriteHeader(201); _=json.NewEncoder(w).Encode(in)
	})
	mux.HandleFunc("/api/drafts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost { http.Error(w,"POST required",405); return }
		var in struct { URL, Caption, AccountID, Tone, Invite string }
		if err:=json.NewDecoder(r.Body).Decode(&in); err!=nil { http.Error(w,"Invalid JSON",400); return }
		if strings.TrimSpace(in.URL)=="" { http.Error(w,"Reel URL is required",400); return }
		a.mu.Lock()
		for _, old := range a.state.Reels { if old.URL==strings.TrimSpace(in.URL) { a.mu.Unlock(); http.Error(w,"This Reel URL is already in the queue",409); return } }
		found:=false
		for _, ac:=range a.state.Accounts { if ac.ID==in.AccountID { found=true; break } }
		if !found && len(a.state.Accounts)>0 { a.mu.Unlock(); http.Error(w,"Choose a saved account profile",400); return }
		d:=Reel{ID:id(),URL:strings.TrimSpace(in.URL),Caption:strings.TrimSpace(in.Caption),AccountID:in.AccountID,Draft:makeDraft(in.Caption,in.Tone,in.Invite),Status:"draft_ready",CreatedAt:time.Now()}
		a.state.Reels=append([]Reel{d},a.state.Reels...); a.saveLocked(); a.mu.Unlock()
		a.notify("ReelMate: a comment draft is ready.\n"+d.URL+"\n\n"+d.Draft)
		w.Header().Set("Content-Type","application/json"); w.WriteHeader(201); _=json.NewEncoder(w).Encode(d)
	})
	mux.HandleFunc("/api/reels/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch { http.Error(w,"PATCH required",405); return }
		rid:=strings.TrimPrefix(r.URL.Path,"/api/reels/")
		var in struct { Status string `json:"status"` }
		if err:=json.NewDecoder(r.Body).Decode(&in); err!=nil { http.Error(w,"Invalid JSON",400); return }
		if in.Status!="draft_ready" && in.Status!="reviewed" && in.Status!="archived" { http.Error(w,"Invalid status",400); return }
		a.mu.Lock(); defer a.mu.Unlock()
		for i:=range a.state.Reels { if a.state.Reels[i].ID==rid { a.state.Reels[i].Status=in.Status; a.saveLocked(); w.WriteHeader(204); return } }
		http.NotFound(w,r)
	})
	mux.HandleFunc("/api/pause", func(w http.ResponseWriter,r *http.Request) {
		if r.Method!=http.MethodPost { http.Error(w,"POST required",405); return }
		a.mu.Lock(); a.state.Paused=!a.state.Paused; paused:=a.state.Paused; a.saveLocked(); a.mu.Unlock()
		if paused { a.notify("ReelMate scheduler paused.") } else { a.notify("ReelMate scheduler resumed.") }
		_ = json.NewEncoder(w).Encode(map[string]bool{"paused":paused})
	})
	return mux
}
func (a *App) scheduler() {
	t := time.NewTicker(25*time.Minute)
	defer t.Stop()
	for range t.C {
		a.mu.Lock()
		a.state.LastTick=time.Now()
		paused:=a.state.Paused
		pending:=0
		for _, r:=range a.state.Reels { if r.Status=="draft_ready" { pending++ } }
		a.saveLocked()
		a.mu.Unlock()
		if !paused { a.notify(fmt.Sprintf("ReelMate 25-minute check: %d draft(s) waiting for review. No comments are posted automatically.",pending)) }
	}
}
func main() {
	path := os.Getenv("DATA_FILE")
	if path == "" {
		path = "data/state.json"
	}
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		_ = os.MkdirAll(dir, 0750)
	}
	app := newApp(path)
	go app.scheduler()
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("ReelMate dashboard on %s", addr)
	log.Fatal(http.ListenAndServe(addr, app.routes()))
}
