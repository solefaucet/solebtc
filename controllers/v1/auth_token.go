package v1

import (
	"fmt"
	"net/http"

	"github.com/freeusd/solebtc/Godeps/_workspace/src/github.com/gin-gonic/gin"
	"github.com/freeusd/solebtc/Godeps/_workspace/src/github.com/satori/go.uuid"
	"github.com/freeusd/solebtc/errors"
	"github.com/freeusd/solebtc/models"
)

type loginPayload struct {
	Email string `json:"email" binding:"required,email"`
}

// dependencies
type (
	loginDependencyGetUserByEmail  func(string) (models.User, *errors.Error)
	loginDependencyCreateAuthToken func(models.AuthToken) *errors.Error
)

// Login logs a existing user in, response with auth token
func Login(
	getUserByEmail loginDependencyGetUserByEmail,
	createAuthToken loginDependencyCreateAuthToken,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		payload := loginPayload{}
		if err := c.BindJSON(&payload); err != nil {
			return
		}

		user, err := getUserByEmail(payload.Email)
		if err != nil {
			switch err.ErrCode {
			case errors.ErrCodeNotFound:
				err.ErrString = fmt.Sprintf("User with email %s does not exist", payload.Email)
				c.AbortWithError(http.StatusNotFound, err)
			default:
				c.AbortWithError(http.StatusInternalServerError, err)
			}
			return
		}

		// create auth token with uuid v4
		authToken := models.AuthToken{
			UserID:    user.ID,
			AuthToken: uuid.NewV4().String(),
		}
		if err := createAuthToken(authToken); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusCreated, authToken)
	}
}