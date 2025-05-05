package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() *httprouter.Router {
	router := httprouter.New()

	router.HandlerFunc(http.MethodGet, "/v1/", app.homeHandler)
	router.HandlerFunc(http.MethodPost, "/v1/signup", app.signupHandler)
	router.HandlerFunc(http.MethodPost, "/v1/signin", app.signinHandler)
	return router
}
