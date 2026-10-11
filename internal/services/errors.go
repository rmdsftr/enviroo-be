package services

import "net/http"

type ServiceError struct {
	Code    int
	Message string
	Data    map[string]interface{}
	ErrCode string
}

func (e *ServiceError) Error() string { return e.Message }

func errNotFound(msg string) *ServiceError {
	return &ServiceError{Code: http.StatusNotFound, Message: msg}
}
func errUnauthorized(msg string) *ServiceError {
	return &ServiceError{Code: http.StatusUnauthorized, Message: msg}
}
func errUnauthorizedCode(msg, errCode string) *ServiceError {
	return &ServiceError{Code: http.StatusUnauthorized, Message: msg, ErrCode: errCode}
}
func errForbidden(msg string) *ServiceError {
	return &ServiceError{Code: http.StatusForbidden, Message: msg}
}
func errBadRequest(msg string) *ServiceError {
	return &ServiceError{Code: http.StatusBadRequest, Message: msg}
}
func errBadRequestCode(msg, errCode string) *ServiceError {
	return &ServiceError{Code: http.StatusBadRequest, Message: msg, ErrCode: errCode}
}
func errConflict(msg string, data map[string]interface{}) *ServiceError {
	return &ServiceError{Code: http.StatusConflict, Message: msg, Data: data}
}
func errInternal(msg string) *ServiceError {
	return &ServiceError{Code: http.StatusInternalServerError, Message: msg}
}
