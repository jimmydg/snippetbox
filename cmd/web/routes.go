package main

import (
	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
	"net/http"
	"snippetbox/ui"
)

func (app *application) routes() http.Handler {
	router := &httprouter.Router{
		RedirectTrailingSlash: true,
		NotFound:              app.customNotFound(),
		RedirectFixedPath:     true,
	}

	fs := http.FileServer(http.FS(ui.Files))
	router.Handler(
		http.MethodGet,
		"/static/*filepath",
		app.neuter(fs),
	)

	router.HandlerFunc(http.MethodGet, "/ping", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	})

	dynamic := alice.New(app.sessionManager.LoadAndSave, noSurf, app.authenticate)

	router.Handler(http.MethodGet, "/", dynamic.ThenFunc(app.home))
	router.Handler(http.MethodGet, "/snippet/view/:id", dynamic.ThenFunc(app.snippetView))
	router.Handler(http.MethodGet, "/user/signup", dynamic.ThenFunc(app.userSignup))
	router.Handler(http.MethodPost, "/user/signup", dynamic.ThenFunc(app.userSignupPost))
	router.Handler(http.MethodGet, "/user/login", dynamic.ThenFunc(app.userLogin))
	router.Handler(http.MethodPost, "/user/login", dynamic.ThenFunc(app.userLoginPost))
	router.Handler(http.MethodGet, "/about", dynamic.ThenFunc(app.about))

	protected := dynamic.Append(app.requireAuth)

	router.Handler(http.MethodGet, "/snippet/create", protected.ThenFunc(app.snippetCreate))
	router.Handler(http.MethodPost, "/snippet/create", protected.ThenFunc(app.snippetCreatePost))
	router.Handler(http.MethodPost, "/user/logout", protected.ThenFunc(app.userLogoutPost))
	router.Handler(http.MethodGet, "/account/view", protected.ThenFunc(app.accountView))

	router.Handler(http.MethodGet, "/account/password/update", protected.ThenFunc(app.accountPasswordUpdate))
	router.Handler(http.MethodPost, "/account/password/update", protected.ThenFunc(app.accountPasswordUpdatePost))

	middleware := alice.New(app.recoverPanic, app.logRequest, secureHeaders)

	return middleware.Then(router)
}
