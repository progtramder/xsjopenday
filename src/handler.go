package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
)

func handleBM(w http.ResponseWriter, r *http.Request) {
	event := r.FormValue("event")
	bmEvent := bmEventList.GetEvent(event)
	if bmEvent == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	//code := r.FormValue("code")
	openId := "testopenid"//GetOpenId(code)
	if openId == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	t, err := template.ParseFiles(systembasePath + "/webroot/html/" + bmEvent.webpage)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	htmlInfo := struct {
		OpenId string
		Event  string
	}{openId, event}
	err = t.Execute(w, htmlInfo)
}

func handleSubmitBM(w http.ResponseWriter, r *http.Request) {
	event := r.FormValue("event")
	bmEvent := bmEventList.GetEvent(event)
	if bmEvent == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	student := r.FormValue("student")
	gender := r.FormValue("gender")
	phone := r.FormValue("phone")
	category := r.FormValue("category")
	s := r.FormValue("session")
	openId := r.FormValue("openid")
	session, _ := strconv.ParseInt(s, 10, 32)
	//category 不是必须属性，由前端js代码决定是否检查
	if openId == "" || student == "" || gender == "" || phone == "" || session < 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	errCode := bmEvent.put(openId, student, gender, phone, category, int(session))
	if errCode == errSuccess {
		register(event, openId, student, gender, phone, category, int(session))
	}
	w.Write([]byte(fmt.Sprintf(`{"errCode":%d,"errMsg":"%s"}`, errCode, Reason(errCode))))
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	event := r.FormValue("event")
	bmEvent := bmEventList.GetEvent(event)
	if bmEvent == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	bmEvent.RLock()
	defer bmEvent.RUnlock()

	type _session struct {
		Desc   string `json:"description"`
		Limit  int    `json:"limit"`
		Number int    `json:"number"`
	}
	status := struct {
		Started  bool       `json:"started"`
		Expired  bool       `json:"expired"`
		Sessions []_session `json:"sessions"`
	}{
		Started: bmEvent.started,
		Expired: bmEvent.Expired(),
	}

	for _, v := range bmEvent.sessions {
		s := _session{
			v.Desc,
			v.Limit,
			v.number,
		}
		status.Sessions = append(status.Sessions, s)
	}

	b, _ := json.Marshal(&status)
	w.Write(b)
}

func handleRegisterInfo(w http.ResponseWriter, r *http.Request) {
	event := r.FormValue("event")
	bmEvent := bmEventList.GetEvent(event)
	if bmEvent == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	openId := r.FormValue("openid")
	if openId == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	bmEvent.RLock()
	defer bmEvent.RUnlock()
	info, ok := bmEvent.has(openId)
	student, gender, phone, category, session := "", "", "", "", ""
	if ok {
		student = info.student
		gender = info.gender
		phone = info.phone
		category = info.category
		session = fmt.Sprintf("%d", info.session)
	}
	w.Write([]byte(fmt.Sprintf(`{"student":"%s","gender":"%s","phone":"%s","category":"%s","session":"%s"}`,
		student, gender, phone, category, session)))
}

// Admin handlers
func checkAuth(r *http.Request) bool {
	h := md5.New()
	io.WriteString(h, privateData.AdminPassword)
	pass := fmt.Sprintf("%x", h.Sum(nil))
	for _, v := range r.Cookies() {
		if v.Name == "admin_password" && v.Value == pass {
			return true
		}
	}

	return false
}

func handleAdmin(w http.ResponseWriter, r *http.Request) {
	adminpage := systembasePath + "/webroot/html/login.html"

	if checkAuth(r) {
		adminpage = systembasePath + "/webroot/html/events.html"
	}

	t, err := template.ParseFiles(adminpage)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	t.Execute(w, nil)
}

// Admin handlers
func handleStartBaoming(w http.ResponseWriter, r *http.Request) {
	event := r.FormValue("event")
	bmEvent := bmEventList.GetEvent(event)
	if bmEvent == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if checkAuth(r) {
		bmEvent.Start()
	}
}

func handleReset(w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("token")
	if token == "" || !tokenPool.get(token) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	err := bmEventList.Reset()
	if err != nil {
		log.Println("Fatal error:", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func handleDevelop(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		if checkAuth(r) {
			developResponse(w, "")
			return
		}

		adminpage := systembasePath + "/webroot/html/develop.html"

		t, err := template.ParseFiles(adminpage)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		t.Execute(w, nil)
	} else {
		token := r.PostFormValue("token")
		if token == "" || !tokenPool.get(token) {
			developResponse(w, fmt.Sprintf("未授权"))
			return
		}

		_, _, err := r.FormFile("uploadfile")
		if err != nil {
			developResponse(w, fmt.Sprintf("上传失败 : %v", err))
			return
		}

		fhs := r.MultipartForm.File["uploadfile"]
		for _, v := range fhs {
			if err := saveUpload(v); err != nil {
				developResponse(w, fmt.Sprintf("上传失败 : %v", err))
				return
			}
		}

		developResponse(w, fmt.Sprintf("上传成功"))
	}
}

func handlGetEvents(w http.ResponseWriter, _ *http.Request) {
	events := struct {
		Data []string `json:"data"`
	}{}

	bmEventList.RLock()
	defer bmEventList.RUnlock()
	for _, v := range bmEventList.events {
		events.Data = append(events.Data, v.name)
	}

	b, _ := json.Marshal(&events)
	w.Write(b)
}
