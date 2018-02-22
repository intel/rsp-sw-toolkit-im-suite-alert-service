package unitTest

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func PerformGet(route string, handlerFunc http.HandlerFunc, t *testing.T) string {
	request, err := http.NewRequest("GET", route, nil)
	if err != nil {
		t.Fatalf("Unable to create new HTTP Request: %s", err.Error())
	}

	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(handlerFunc)

	handler.ServeHTTP(recorder, request)
	if recorder.Code != 200 {
		t.Errorf("Success expected: %d", recorder.Code) //Uh-oh this means our test failed
	}

	result := recorder.Body.String()

	return result
}
