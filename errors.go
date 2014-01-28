package main

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
)

type UserFriendlyError interface {
	error
	UserFriendlyError() string
}

var _ UserFriendlyError = (*stringUserFriendlyError)(nil)

type stringUserFriendlyError struct {
	error
	uerr string
}

func (s *stringUserFriendlyError) UserFriendlyError() string {
	return s.uerr
}

func ErrStartAndGoal(e error) *stringUserFriendlyError {
	return &stringUserFriendlyError{e, "I could not find where the wiki is in the intertubes."}
}

func ErrMalformedQuery(e error) *stringUserFriendlyError {
	return &stringUserFriendlyError{e, "The stuff you typed in the URI I don't understand."}
}

func ErrGameMarshal(e error) *stringUserFriendlyError {
	return &stringUserFriendlyError{e,"I could not save th1s game! M4ybe m_ disks a-re f-f-f---aulll-.."}
}

func ErrGetGameSession(e error) *stringUserFriendlyError {
	return &stringUserFriendlyError{e, "Tried to get your game session but failed. Maybe you threw away your cookies? Who would do that to his cookies?!"}
}

func ErrGetGame(e error) *stringUserFriendlyError {
	return &stringUserFriendlyError{e, "I failed to retrieve this game, maybe there's something wrong?"}
}

func ErrPlayerLoad(e error) *stringUserFriendlyError {
	return &stringUserFriendlyError{e, "Loading your player profile for the game failed."}
}

func ErrNoSuchGame(gameId string) *stringUserFriendlyError {
	return &stringUserFriendlyError{
		fmt.Errorf("Requested invalid game %s", gameId),
		fmt.Sprintf("There does not seem to be such a game as %s. Maybe your game expired or there's an error on our side.", gameId),
	}
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

	if e, ok := err.(UserFriendlyError); ok && e != nil {
		// We have recovered an error expected by the programmer,
		// handle it gracefully.
		logError(e, r)

		templates.ExecuteTemplate(w, "error.html", struct {
			ErrorMessage               string
			UnderstandableErrorMessage string
		}{e.Error(), e.UserFriendlyError()})

	} else if ok && e == nil {
		// There is an error but the programmer failed to supply
		// a valid object. This is a programming error and cannot
		// be handled gracefully.
		logError(err, r)

		panic("Invalid error")

	} else if err != nil {
		// What we have recovered is no error but non-nil, this is
		// unexpected and can't be handled most user friendly.
		logError(err, r)

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
