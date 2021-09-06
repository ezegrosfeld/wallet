package user

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ezegrosfeld/wallet/users/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockedService struct {
	mock.Mock
}

func (s *mockedService) Create(ctx context.Context, username, password string) (*domain.User, error) {
	args := s.Called(username, password)
	return args.Get(0).(*domain.User), args.Error(1)
}

func (s *mockedService) Login(ctx context.Context, username, password string) (*domain.User, error) {
	args := s.Called(username, password)
	return args.Get(0).(*domain.User), args.Error(1)
}

func createServer(s *mockedService) *gin.Engine {
	h := NewHandler(s)

	router := gin.Default()

	router.POST("/users", h.Create())
	router.POST("/login", h.Login())

	return router
}

func createRequest(method, endpoint, body string) (*http.Request, *httptest.ResponseRecorder) {
	r := httptest.NewRequest(method, endpoint, strings.NewReader(body))
	r.Header.Add("Content-Type", "application/json")
	rw := httptest.NewRecorder()

	return r, rw
}

func TestCreateHandler(t *testing.T) {
	ms := &mockedService{}

	user := &domain.User{
		Username: "username",
		Password: "password",
	}

	ms.On("Create", user.Username, user.Password).Return(user, nil)

	s := createServer(ms)

	r, rw := createRequest("POST", "/users", `{"username": "username", "password": "password"}`)

	s.ServeHTTP(rw, r)

	type response struct {
		domain.User
	}

	res := new(response)

	err := json.Unmarshal(rw.Body.Bytes(), &res)

	tk := rw.Header().Get("Set-Cookie")

	a := assert.New(t)
	a.NoError(err)
	a.Equal(http.StatusCreated, rw.Code)
	a.Equal(user.Username, res.Username)
	a.NotEmpty(t, tk)
}

func TestCreateMissingField(t *testing.T) {
	ms := &mockedService{}

	user := &domain.User{
		Username: "username",
		Password: "password",
	}

	ms.On("Create", user.Username, user.Password).Return(user, nil)

	s := createServer(ms)

	r, rw := createRequest("POST", "/users", `{"password": "password"}`)

	s.ServeHTTP(rw, r)

	a := assert.New(t)
	a.Equal(http.StatusBadRequest, rw.Code)
}

func TestCreateErrorWithService(t *testing.T) {
	ms := &mockedService{}

	user := &domain.User{
		Username: "username",
		Password: "password",
	}

	ms.On("Create", user.Username, user.Password).Return(user, fmt.Errorf("Something went wrong"))

	s := createServer(ms)

	r, rw := createRequest("POST", "/users", `{"username": "username", "password": "password"}`)

	s.ServeHTTP(rw, r)

	type response struct {
		Error string `json:"error"`
	}

	res := new(response)

	err := json.Unmarshal(rw.Body.Bytes(), &res)

	a := assert.New(t)
	a.NoError(err)
	a.Equal(http.StatusInternalServerError, rw.Code)
	a.Equal("Something went wrong", res.Error)
}

func TestHandlerCreateWithConflict(t *testing.T) {
	ms := &mockedService{}

	user := &domain.User{
		Username: "username",
		Password: "password",
	}

	ms.On("Create", user.Username, user.Password).Return(user, ErrConflict)

	s := createServer(ms)

	r, rw := createRequest("POST", "/users", `{"username": "username", "password": "password"}`)

	s.ServeHTTP(rw, r)

	type response struct {
		Error string `json:"error"`
	}

	res := new(response)

	err := json.Unmarshal(rw.Body.Bytes(), &res)

	a := assert.New(t)
	a.NoError(err)
	a.Equal(http.StatusConflict, rw.Code)
}

func TestLoginHandler(t *testing.T) {
	ms := &mockedService{}

	user := &domain.User{
		Username: "username",
		Password: "password",
	}

	ms.On("Login", user.Username, user.Password).Return(user, nil)

	s := createServer(ms)

	r, rw := createRequest("POST", "/login", `{"username": "username", "password": "password"}`)

	s.ServeHTTP(rw, r)

	type response struct {
		domain.User
	}

	res := new(response)

	err := json.Unmarshal(rw.Body.Bytes(), &res)

	tk := rw.Header().Get("Set-Cookie")

	a := assert.New(t)
	a.NoError(err)
	a.Equal(http.StatusCreated, rw.Code)
	a.NotEmpty(tk)
}

func TestLoginHandlerBadRquest(t *testing.T) {
	ms := &mockedService{}

	user := &domain.User{
		Username: "username",
		Password: "password",
	}

	ms.On("Login", user.Username, user.Password).Return(user, nil)

	s := createServer(ms)

	r, rw := createRequest("POST", "/login", `{"username": "username"}`)

	s.ServeHTTP(rw, r)

	a := assert.New(t)
	a.Equal(http.StatusBadRequest, rw.Code)
}

func TestLoginHandlerUserNotFound(t *testing.T) {
	ms := &mockedService{}

	user := &domain.User{
		Username: "username",
		Password: "password",
	}

	ms.On("Login", user.Username, user.Password).Return(user, ErrNotFound)

	s := createServer(ms)

	r, rw := createRequest("POST", "/login", `{"username": "username", "password": "password"}`)

	s.ServeHTTP(rw, r)

	a := assert.New(t)
	a.Equal(http.StatusNotFound, rw.Code)
}

func TestLoginHandlerPasswordIncorrect(t *testing.T) {
	ms := &mockedService{}

	user := &domain.User{
		Username: "username",
		Password: "password",
	}

	ms.On("Login", user.Username, user.Password).Return(user, ErrWrongPassword)

	s := createServer(ms)

	r, rw := createRequest("POST", "/login", `{"username": "username", "password": "password"}`)

	s.ServeHTTP(rw, r)

	a := assert.New(t)
	a.Equal(http.StatusUnauthorized, rw.Code)
}
