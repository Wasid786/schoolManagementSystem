package main

import (
	"fmt"
	"net/http"
)

func (app *application) homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Homepage handler!")
}
