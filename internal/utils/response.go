package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrResp struct {
	Error string `json:"error"`
}

type SuccResp struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func ErrJSON(c *gin.Context, code int, msg string) {
	c.JSON(code, ErrResp{Error: msg})
}

func SuccJSON(c *gin.Context, msg string, data interface{}) {
	c.JSON(http.StatusOK, SuccResp{Message: msg, Data: data})
}
