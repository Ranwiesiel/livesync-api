
package response

import "github.com/gin-gonic/gin"

type APIResponse struct {
    Success bool   `json:"success"`
    Data    any    `json:"data"`
    Error   any    `json:"error"`
}

func Success(c *gin.Context, status int, data any) {
    c.JSON(status, APIResponse{
        Success: true,
        Data:    data,
        Error:   nil,
    })
}

func Failure(c *gin.Context, status int, message string) {
    c.JSON(status, APIResponse{
        Success: false,
        Data:    nil,
        Error:   gin.H{"message": message},
    })
}
