package services

import "net/http"

type ServiceError struct {
	Code    int
	Message string
	Data    map[string]interface{}
}

func (e *ServiceError) Error() string { return e.Message }

func errNotFound(msg string) *ServiceError {
	return &ServiceError{Code: http.StatusNotFound, Message: msg}
}
func errForbidden(msg string) *ServiceError {
	return &ServiceError{Code: http.StatusForbidden, Message: msg}
}
func errBadRequest(msg string) *ServiceError {
	return &ServiceError{Code: http.StatusBadRequest, Message: msg}
}
func errConflict(msg string, data map[string]interface{}) *ServiceError {
	return &ServiceError{Code: http.StatusConflict, Message: msg, Data: data}
}
func errInternal(msg string) *ServiceError {
	return &ServiceError{Code: http.StatusInternalServerError, Message: msg}
}
