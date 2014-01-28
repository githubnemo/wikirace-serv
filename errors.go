package main

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
)

type ErrStartAndGoal error

type ErrMalformedQuery error

type ErrGameMarshal error

type ErrGetGameSession error

type ErrGetGame error

type ErrNoSuchGame string

func (e ErrNoSuchGame) Error() string {
	return "Game " + string(e) + " could not be found in game store."
}

// Translate error object to user friendly message if possible.
func userFriendlyError(e error) string {
	switch e.(type) {
	case ErrStartAndGoal:
		return "I could not find where the wiki is in the intertubes."
	case ErrMalformedQuery:
		return "The stuff you typed in the URI I don't understand."
	case ErrGameMarshal:
		return "I could not save th1s game! M4ybe m_ disks a-re f-f-f---aulll-.."
	case ErrGetGameSession:
		return "Tried to get your game session but failed. Maybe you threw away your cookies? Who would do that to his cookies?!"
	case ErrNoSuchGame:
		return "Pretty much what it says, there does not seem to be such a game. Maybe your game expired or there's an error on our side."
	case ErrGetGame:
		return "I failed to retrieve this game, maybe there's something wrong?"
	}

	// We don't know this error. Panic to let the commonErrorHandler handle
	// this unknown error.
	panic(e)
}

func logError(err interface{}, r *http.Request) {
	log.Println(
		"panic catched:", err,
		"\nRequest data:", r,
		"\nStack:", string(debug.Stack()))
}

func commonErrorHandler(w http.ResponseWriter, r *http.Request) {
	if err := recover(); err != nil {
		w.WriteHeader(401)

		fmt.Fprintf(w, "Oh...:(\n\n")

		if e, ok := err.(error); ok {
			w.Write([]byte(e.Error()))
			w.Write([]byte{'\n', '\n'})
			w.Write(debug.Stack())
		} else {
			fmt.Fprintf(w, "%s\n\n%s\n", err, debug.Stack())
		}
	}
}

func userFriendlyErrorHandler(w http.ResponseWriter, r *http.Request) {
	// In case the error can't be handled in here fall back to the
	// common error handler.
	defer commonErrorHandler(w, r)

	err := recover()

	if err == nil {
		return
	}

	logError(err, r)

	if e, ok := err.(error); ok && e != nil {
		understandableErrorMessage := userFriendlyError(e)

		templates.ExecuteTemplate(w, "error.html", struct {
			ErrorMessage               string
			UnderstandableErrorMessage string
		}{e.Error(), understandableErrorMessage})

	} else {
		// Re-panic as we haven't handled the error yet.
		panic(err)
	}
}

// Wrap error handling chain around a http.HandlerFunc.
//
// First errors are tried to be resolved to meaningful user friendly
// messages in userFriendlyErrorHandler. In case that fails,
// a second error handler, the commonErrorHandler kicks in and reports some
// basic crash report.
//
func errorHandler(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer userFriendlyErrorHandler(w, r)
		f(w, r)
	}
}
