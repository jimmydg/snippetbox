package main

import (
	"errors"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"snippetbox/internal/models"
	"snippetbox/internal/validator"
	"strconv"
	"strings"
)

type snippetCreateForm struct {
	Title               string `form:"title"`
	Content             string `form:"content"`
	Expires             int    `form:"expires"`
	validator.Validator `form:"-"`
}

type userSignupForm struct {
	Name                string `form:"name"`
	Email               string `form:"email"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}

type userLoginForm struct {
	Email               string `form:"email"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}

type accountPasswordUpdateForm struct {
	CurrentPassword         string `form:"currentPassword"`
	NewPassword             string `form:"newPassword"`
	NewPasswordConfirmation string `form:"newPasswordConfirmation"`
	validator.Validator     `form:"-"`
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	snippets, err := app.snippets.Latest()
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := app.newTemplateData(r)
	data.Snippets = snippets

	app.render(w, r, http.StatusOK, "home.gohtml", data)
}

func (app *application) about(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, http.StatusOK, "about.gohtml", app.newTemplateData(r))
}

func (app *application) snippetView(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	id, err := strconv.Atoi(params.ByName("id"))
	if err != nil || id < 1 {
		app.notFound(w)
		return
	}

	snippet, err := app.snippets.Get(id)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			app.notFound(w)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	data := app.newTemplateData(r)
	data.Snippet = snippet

	app.render(w, r, http.StatusOK, "view.gohtml", data)
}

func (app *application) snippetCreate(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = snippetCreateForm{Expires: 365}
	app.render(w, r, http.StatusOK, "create.gohtml", data)
}

func (app *application) snippetCreatePost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	var form snippetCreateForm

	err = app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Title), "title", "This field cannot be blank")
	form.CheckField(validator.MaxChars(form.Title, 100), "title", "This field cannot be more than 100 chars")
	form.CheckField(validator.NotBlank(form.Content), "content", "This field cannot be blank")
	form.CheckField(validator.PermittedValue(form.Expires, 1, 7, 365), "expires", "This field must equal 1, 7 or 365")

	if !form.Valid() {
		app.render(w, r, http.StatusUnprocessableEntity, "create.gohtml", app.newTemplateDataForm(form, r))
		return
	}

	snippetModel, err := models.NewSnippetInsertModel(app.db)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	id, err := snippetModel.Insert(form.Title, form.Content, form.Expires)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Snippet successfully created!")

	http.Redirect(w, r, fmt.Sprintf("/snippet/view/%d", id), http.StatusSeeOther)
}

func (app *application) downloadHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "ui/static/tmp.zip")
}

// neuter disables directory listings
func (app *application) neuter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			app.notFound(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (app *application) customNotFound() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.notFound(w)
	})
}

func (app *application) userSignup(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = userSignupForm{}
	app.render(w, r, http.StatusOK, "signup.gohtml", data)
}

func (app *application) userSignupPost(w http.ResponseWriter, r *http.Request) {
	var form userSignupForm

	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Name), "name", "This field cannot be blank")
	form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
	form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email", "This field must be a valid email address")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")
	form.CheckField(validator.MinChars(form.Password, 8), "password", "This field must be at least 8 characters long")

	if !form.Valid() {
		app.render(w, r, http.StatusUnprocessableEntity, "signup.gohtml", app.newTemplateDataForm(form, r))
		return
	}

	err = app.users.Insert(form.Name, form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrDuplicateMail) {
			form.AddFieldError("email", "Email already in use")
			app.render(w, r, http.StatusUnprocessableEntity, "signup.gohtml", app.newTemplateDataForm(form, r))
			return
		}
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Signup was successful. You can now login.")

	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

func (app *application) userLogin(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = userLoginForm{}
	app.render(w, r, http.StatusOK, "login.gohtml", data)
}

func (app *application) userLoginPost(w http.ResponseWriter, r *http.Request) {
	var form userLoginForm

	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
	form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email", "This field must be a valid email address")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")

	if !form.Valid() {
		app.render(w, r, http.StatusUnprocessableEntity, "login.gohtml", app.newTemplateDataForm(form, r))
		return
	}

	id, err := app.users.Auth(form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrInvalidCreds) {
			form.AddNonFieldError("Email or password is incorrect")

			app.render(w, r, http.StatusUnprocessableEntity, "login.gohtml", app.newTemplateDataForm(form, r))
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	// it's a good idea to change the token on state change. Login in this case.
	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Put(r.Context(), authenticatedUserId, id)

	// retrieve and remove session value
	path := app.sessionManager.PopString(r.Context(), redirectAfterLogin)
	if path != "" {
		http.Redirect(w, r, path, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/snippet/create", http.StatusSeeOther)
}

func (app *application) userLogoutPost(w http.ResponseWriter, r *http.Request) {
	err := app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Remove(r.Context(), authenticatedUserId)
	app.sessionManager.Put(r.Context(), "flash", "Logged out.")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) accountView(w http.ResponseWriter, r *http.Request) {
	userId := app.sessionManager.GetInt(r.Context(), authenticatedUserId)
	user, err := app.users.Get(userId)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			http.Redirect(w, r, "/user/login", http.StatusSeeOther)
			return
		}
		app.serverError(w, r, err)
		return
	}

	data := app.newTemplateData(r)
	data.User = user
	app.render(w, r, http.StatusOK, "account.gohtml", data)
}

func (app *application) accountPasswordUpdate(w http.ResponseWriter, r *http.Request) {
	var form accountPasswordUpdateForm

	app.render(w, r, http.StatusOK, "password.gohtml", app.newTemplateDataForm(form, r))
}

func (app *application) accountPasswordUpdatePost(w http.ResponseWriter, r *http.Request) {
	var form accountPasswordUpdateForm

	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// not empty
	form.CheckField(validator.NotBlank(form.CurrentPassword), "currentPassword", "This field is required.")
	form.CheckField(validator.NotBlank(form.NewPassword), "newPassword", "This field is required.")
	form.CheckField(validator.NotBlank(form.NewPasswordConfirmation), "newPasswordConfirmation", "This field is required.")
	// minChars
	form.CheckField(validator.MinChars(form.NewPassword, 8), "newPassword", "Must be at least 8 characters long.")
	// old/new password equal
	form.CheckField(form.NewPassword == form.NewPasswordConfirmation, "newPasswordMismatch", "New password are not equal")

	if !form.Valid() {
		app.render(w, r, http.StatusUnprocessableEntity, "password.gohtml", app.newTemplateDataForm(form, r))
		return
	}

	err = app.users.PasswordUpdate(
		app.sessionManager.GetInt(r.Context(), authenticatedUserId),
		form.CurrentPassword,
		form.NewPassword,
	)

	if err != nil {
		if errors.Is(err, models.ErrInvalidCreds) {
			form.AddFieldError("currentPassword", "Current password is incorrect")
			app.render(w, r, http.StatusUnprocessableEntity, "password.gohtml", app.newTemplateDataForm(form, r))
			return
		}
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Your password has been updated!")

	http.Redirect(w, r, "/account/view", http.StatusSeeOther)
}
